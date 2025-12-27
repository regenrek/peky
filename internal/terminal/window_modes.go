package terminal

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

type CopyMode struct {
	Active bool

	// Absolute coords in combined buffer:
	// 0..ScrollbackLen-1 == scrollback lines
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
	if w.IsAltScreen() {
		return
	}
	w.stateMu.Lock()
	w.ScrollbackMode = true
	w.stateMu.Unlock()
	w.markDirty()
}

func (w *Window) ExitScrollback() {
	if w == nil {
		return
	}
	w.stateMu.Lock()
	w.ScrollbackOffset = 0
	if w.CopyMode == nil || !w.CopyMode.Active {
		w.ScrollbackMode = false
	}
	w.stateMu.Unlock()
	w.markDirty()
}

func (w *Window) ScrollUp(lines int) {
	if w == nil || lines <= 0 {
		return
	}
	if w.IsAltScreen() {
		return
	}
	sbLen := w.ScrollbackLen()
	if sbLen <= 0 {
		return
	}

	w.stateMu.Lock()
	w.ScrollbackMode = true
	w.ScrollbackOffset = clampInt(w.ScrollbackOffset+lines, 0, sbLen)
	w.stateMu.Unlock()
	w.markDirty()
}

func (w *Window) ScrollDown(lines int) {
	if w == nil || lines <= 0 {
		return
	}
	if w.IsAltScreen() {
		return
	}

	w.stateMu.Lock()
	w.ScrollbackOffset -= lines
	if w.ScrollbackOffset < 0 {
		w.ScrollbackOffset = 0
	}
	if w.ScrollbackOffset == 0 && (w.CopyMode == nil || !w.CopyMode.Active) {
		w.ScrollbackMode = false // auto-exit
	}
	w.stateMu.Unlock()
	w.markDirty()
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
	if w == nil || w.IsAltScreen() {
		return
	}
	sbLen := w.ScrollbackLen()
	w.stateMu.Lock()
	w.ScrollbackMode = true
	w.ScrollbackOffset = sbLen
	w.stateMu.Unlock()
	w.markDirty()
}

func (w *Window) ScrollToBottom() {
	w.ExitScrollback()
}

func (w *Window) EnterCopyMode() {
	if w == nil || w.IsAltScreen() {
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

func (w *Window) adjustToNonContinuation(absY, x, dir int) int {
	if w == nil {
		return x
	}
	if x < 0 {
		x = 0
	}
	if x >= w.cols {
		x = maxInt(0, w.cols-1)
	}
	if dir == 0 {
		dir = -1
	}

	w.termMu.Lock()
	defer w.termMu.Unlock()
	term := w.term
	if term == nil {
		return x
	}
	sbLen := term.ScrollbackLen()

	for x > 0 && x < w.cols {
		cell := cellAtAbs(term, sbLen, w.rows, x, absY)
		if cell == nil || cell.Width != 0 {
			return x
		}
		x += dir
		if x < 0 {
			return 0
		}
		if x >= w.cols {
			return maxInt(0, w.cols-1)
		}
	}
	return x
}

func (w *Window) extractText(startX, startAbsY, endX, endAbsY int) string {
	if w == nil {
		return ""
	}
	if startAbsY > endAbsY || (startAbsY == endAbsY && startX > endX) {
		startX, endX = endX, startX
		startAbsY, endAbsY = endAbsY, startAbsY
	}

	width := w.cols
	if width <= 0 {
		return ""
	}

	w.termMu.Lock()
	defer w.termMu.Unlock()
	term := w.term
	if term == nil {
		return ""
	}

	sbLen := term.ScrollbackLen()
	total := sbLen + w.rows
	if total <= 0 {
		return ""
	}

	startAbsY = clampInt(startAbsY, 0, total-1)
	endAbsY = clampInt(endAbsY, 0, total-1)
	startX = clampInt(startX, 0, maxInt(0, width-1))
	endX = clampInt(endX, 0, maxInt(0, width-1))

	var out strings.Builder

	for y := startAbsY; y <= endAbsY; y++ {
		x0 := 0
		x1 := width - 1
		if y == startAbsY {
			x0 = startX
		}
		if y == endAbsY {
			x1 = endX
		}
		x0 = clampInt(x0, 0, width-1)
		x1 = clampInt(x1, 0, width-1)

		var line strings.Builder
		for x := x0; x <= x1; x++ {
			cell := cellAtAbs(term, sbLen, w.rows, x, y)
			if cell != nil && cell.Width == 0 {
				continue
			}
			if cell != nil && cell.Content != "" {
				line.WriteString(cell.Content)
			} else {
				line.WriteByte(' ')
			}
		}

		out.WriteString(strings.TrimRight(line.String(), " "))
		if y != endAbsY {
			out.WriteByte('\n')
		}
	}

	return strings.TrimRight(out.String(), "\n")
}

func cellAtAbs(term vtEmulator, sbLen, rows, x, absY int) *uv.Cell {
	if absY < sbLen {
		line := term.ScrollbackLine(absY)
		if line == nil || x < 0 || x >= len(line) {
			return nil
		}
		return &line[x]
	}
	screenY := absY - sbLen
	if screenY < 0 || screenY >= rows {
		return nil
	}
	return term.CellAt(x, screenY)
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
