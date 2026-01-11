package terminal

type CopyMode struct {
	Active bool

	// Absolute coords in combined buffer:
	// 0..ScrollbackLen-1 == scrollback rows
	// ScrollbackLen..ScrollbackLen+Rows-1 == current screen
	CursorX    int
	CursorAbsY int

	Selecting    bool
	SelStartX    int
	SelStartAbsY int
	SelEndX      int
	SelEndAbsY   int
}

func (w *Window) CopyModeActive() bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.CopyMode != nil && w.CopyMode.Active
}

func (w *Window) CopySelectionActive() bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.CopyMode != nil && w.CopyMode.Active && w.CopyMode.Selecting
}

func (w *Window) CopySelectionFromMouseActive() bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.CopyMode != nil && w.CopyMode.Active && w.CopyMode.Selecting && w.mouseSel.fromMouse
}

func (w *Window) ScrollbackModeActive() bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.ScrollbackMode
}

func (w *Window) GetScrollbackOffset() int {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.ScrollbackOffset
}

func (w *Window) ScrollbackLen() int {
	if w == nil {
		return 0
	}
	w.termMu.Lock()
	defer w.termMu.Unlock()
	if w.term == nil {
		return 0
	}
	return w.term.ScrollbackLen()
}

func (w *Window) IsAltScreen() bool {
	if w == nil {
		return false
	}
	if w.altScreen.Load() {
		return true
	}
	w.termMu.Lock()
	term := w.term
	w.termMu.Unlock()
	if term == nil {
		return false
	}
	return term.IsAltScreen()
}

func (w *Window) onScrollbackGrew(delta int) {
	if w == nil || delta <= 0 {
		return
	}
	w.stateMu.Lock()
	defer w.stateMu.Unlock()

	// If user is scrolled up, keep the same viewport anchored by increasing offset.
	if w.ScrollbackOffset > 0 {
		w.ScrollbackOffset += delta
		// Clamp later via clampViewState().
	}
}

func (w *Window) clampViewState() {
	if w == nil {
		return
	}
	sbLen := w.ScrollbackLen()
	total := sbLen + w.rows
	if total <= 0 {
		total = 0
	}

	w.stateMu.Lock()
	defer w.stateMu.Unlock()

	if w.ScrollbackOffset < 0 {
		w.ScrollbackOffset = 0
	}
	if w.ScrollbackOffset > sbLen {
		w.ScrollbackOffset = sbLen
	}

	// Auto-exit scrollback mode when back to live view (and not in copy mode).
	if w.ScrollbackOffset == 0 && (w.CopyMode == nil || !w.CopyMode.Active) {
		w.ScrollbackMode = false
	}

	if w.CopyMode != nil && w.CopyMode.Active {
		if w.CopyMode.CursorX < 0 {
			w.CopyMode.CursorX = 0
		}
		if w.CopyMode.CursorX >= w.cols {
			w.CopyMode.CursorX = maxInt(0, w.cols-1)
		}
		if total > 0 {
			if w.CopyMode.CursorAbsY < 0 {
				w.CopyMode.CursorAbsY = 0
			}
			if w.CopyMode.CursorAbsY >= total {
				w.CopyMode.CursorAbsY = total - 1
			}
		}
		w.ensureCopyCursorVisibleLocked(sbLen)
	}
}

func (w *Window) EnterScrollback() {
	if w == nil {
		return
	}
	w.stateMu.Lock()
	changed := !w.ScrollbackMode
	w.ScrollbackMode = true
	w.stateMu.Unlock()
	if changed {
		w.markDirty()
	}
}

func (w *Window) ExitScrollback() {
	if w == nil {
		return
	}
	w.stateMu.Lock()
	oldOffset := w.ScrollbackOffset
	oldMode := w.ScrollbackMode
	w.ScrollbackOffset = 0
	if w.CopyMode == nil || !w.CopyMode.Active {
		w.ScrollbackMode = false
	}
	changed := oldOffset != w.ScrollbackOffset || oldMode != w.ScrollbackMode
	w.stateMu.Unlock()
	if changed {
		w.markDirty()
	}
}

func (w *Window) ScrollUp(lines int) {
	if w == nil || lines <= 0 {
		return
	}
	sbLen := w.ScrollbackLen()
	if sbLen <= 0 {
		return
	}

	w.stateMu.Lock()
	oldOffset := w.ScrollbackOffset
	oldMode := w.ScrollbackMode
	newOffset := clampInt(w.ScrollbackOffset+lines, 0, sbLen)
	w.ScrollbackMode = true
	w.ScrollbackOffset = newOffset
	changed := oldOffset != w.ScrollbackOffset || oldMode != w.ScrollbackMode
	w.stateMu.Unlock()
	if changed {
		w.markDirty()
	}
}

func (w *Window) ScrollDown(lines int) {
	if w == nil || lines <= 0 {
		return
	}

	w.stateMu.Lock()
	oldOffset := w.ScrollbackOffset
	oldMode := w.ScrollbackMode
	w.ScrollbackOffset -= lines
	if w.ScrollbackOffset < 0 {
		w.ScrollbackOffset = 0
	}
	if w.ScrollbackOffset > 0 {
		w.ScrollbackMode = true
	}
	if w.ScrollbackOffset == 0 && (w.CopyMode == nil || !w.CopyMode.Active) {
		w.ScrollbackMode = false // auto-exit
	}
	changed := oldOffset != w.ScrollbackOffset || oldMode != w.ScrollbackMode
	w.stateMu.Unlock()
	if changed {
		w.markDirty()
	}
}

func (w *Window) PageUp() {
	if w == nil {
		return
	}
	step := maxInt(1, w.rows-1)
	w.ScrollUp(step)
}

func (w *Window) PageDown() {
	if w == nil {
		return
	}
	step := maxInt(1, w.rows-1)
	w.ScrollDown(step)
}

func (w *Window) ScrollToTop() {
	if w == nil {
		return
	}
	sbLen := w.ScrollbackLen()
	w.stateMu.Lock()
	oldOffset := w.ScrollbackOffset
	oldMode := w.ScrollbackMode
	w.ScrollbackMode = true
	w.ScrollbackOffset = sbLen
	changed := oldOffset != w.ScrollbackOffset || oldMode != w.ScrollbackMode
	w.stateMu.Unlock()
	if changed {
		w.markDirty()
	}
}

func (w *Window) ScrollToBottom() {
	w.ExitScrollback()
}

func (w *Window) EnterCopyMode() {
	w.enterCopyMode(false)
}

func (w *Window) enterCopyMode(allowAltScreen bool) {
	if w == nil {
		return
	}
	if !allowAltScreen && w.IsAltScreen() {
		return
	}

	w.termMu.Lock()
	term := w.term
	if term == nil {
		w.termMu.Unlock()
		return
	}
	sbLen := term.ScrollbackLen()
	cur := term.CursorPosition()
	w.termMu.Unlock()

	// Determine initial cursor position.
	w.stateMu.Lock()
	offset := w.ScrollbackOffset
	topAbsY := sbLen - clampInt(offset, 0, sbLen)
	if topAbsY < 0 {
		topAbsY = 0
	}
	total := sbLen + w.rows

	cm := &CopyMode{Active: true}

	if offset > 0 || w.ScrollbackMode {
		// In scrollback view: start at bottom of viewport.
		cm.CursorAbsY = clampInt(topAbsY+w.rows-1, 0, maxInt(0, total-1))
		cm.CursorX = 0
	} else {
		// Live view: start at current terminal cursor.
		cm.CursorAbsY = clampInt(sbLen+cur.Y, 0, maxInt(0, total-1))
		cm.CursorX = clampInt(cur.X, 0, maxInt(0, w.cols-1))
	}

	cm.SelStartX, cm.SelStartAbsY = cm.CursorX, cm.CursorAbsY
	cm.SelEndX, cm.SelEndAbsY = cm.CursorX, cm.CursorAbsY

	w.CopyMode = cm
	w.mouseSel.clearSelectionFlags()
	w.ensureCopyCursorVisibleLocked(sbLen)
	w.stateMu.Unlock()

	w.markDirty()
}

func (w *Window) ExitCopyMode() {
	if w == nil {
		return
	}
	w.stateMu.Lock()
	w.CopyMode = nil
	w.mouseSel.clearSelectionFlags()
	// Auto-exit scrollback mode if at live view.
	if w.ScrollbackOffset == 0 {
		w.ScrollbackMode = false
	}
	w.stateMu.Unlock()
	w.markDirty()
}

func (w *Window) CopyToggleSelect() {
	if w == nil {
		return
	}
	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return
	}
	w.mouseSel.clearSelectionFlags()
	if !cm.Selecting {
		cm.Selecting = true
		cm.SelStartX, cm.SelStartAbsY = cm.CursorX, cm.CursorAbsY
		cm.SelEndX, cm.SelEndAbsY = cm.CursorX, cm.CursorAbsY
	} else {
		// Exit selection.
		cm.Selecting = false
		cm.SelStartX, cm.SelStartAbsY = cm.CursorX, cm.CursorAbsY
		cm.SelEndX, cm.SelEndAbsY = cm.CursorX, cm.CursorAbsY
	}
	w.stateMu.Unlock()
	w.markDirty()
}

func (w *Window) CopyMove(dx, dy int) {
	if w == nil {
		return
	}

	sbLen := w.ScrollbackLen()
	total := sbLen + w.rows
	if total <= 0 {
		return
	}

	// Snapshot current cursor.
	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return
	}
	newX := clampInt(cm.CursorX+dx, 0, maxInt(0, w.cols-1))
	newAbsY := clampInt(cm.CursorAbsY+dy, 0, total-1)
	w.stateMu.Unlock()

	// Avoid landing on continuation cells (wide chars).
	dir := -1
	if dx > 0 {
		dir = 1
	}
	newX = w.adjustToNonContinuation(newAbsY, newX, dir)

	w.stateMu.Lock()
	cm = w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return
	}
	cm.CursorX = newX
	cm.CursorAbsY = newAbsY
	if cm.Selecting {
		cm.SelEndX = newX
		cm.SelEndAbsY = newAbsY
	}
	w.ensureCopyCursorVisibleLocked(sbLen)
	w.stateMu.Unlock()

	w.markDirty()
}

func (w *Window) CopyPageUp()   { w.CopyMove(0, -maxInt(1, w.rows-1)) }
func (w *Window) CopyPageDown() { w.CopyMove(0, maxInt(1, w.rows-1)) }

func (w *Window) CopyYankText() string {
	if w == nil {
		return ""
	}

	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return ""
	}
	startX, startY := cm.CursorX, cm.CursorAbsY
	endX, endY := cm.CursorX, cm.CursorAbsY
	if cm.Selecting {
		startX, startY = cm.SelStartX, cm.SelStartAbsY
		endX, endY = cm.SelEndX, cm.SelEndAbsY
	}
	w.stateMu.Unlock()

	return w.extractText(startX, startY, endX, endY)
}

func (w *Window) ensureCopyCursorVisibleLocked(sbLen int) {
	// stateMu must already be held.
	if w.CopyMode == nil || !w.CopyMode.Active {
		return
	}
	offset := clampInt(w.ScrollbackOffset, 0, sbLen)
	topAbsY := sbLen - offset
	if topAbsY < 0 {
		topAbsY = 0
	}

	total := sbLen + w.rows
	if total <= 0 {
		return
	}
	bottomAbsY := topAbsY + (w.rows - 1)
	if bottomAbsY > total-1 {
		bottomAbsY = total - 1
	}

	curY := w.CopyMode.CursorAbsY
	if curY < topAbsY {
		topAbsY = curY
	} else if curY > bottomAbsY {
		topAbsY = curY - (w.rows - 1)
	}

	if topAbsY < 0 {
		topAbsY = 0
	}
	if topAbsY > sbLen {
		topAbsY = sbLen
	}

	w.ScrollbackOffset = clampInt(sbLen-topAbsY, 0, sbLen)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
