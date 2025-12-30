package terminal

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

const (
	mouseModeX10 uint32 = 1 << iota
	mouseModeNormal
	mouseModeHighlight
	mouseModeButtonEvent
	mouseModeAnyEvent
	mouseModeSGR
)

const (
	mouseWheelLinesDefault = 3
)

const mouseSelectionCopiedToast = "Selection copied to clipboard"

func (w *Window) mouseWheelStep(mod uv.KeyMod) int {
	if w == nil {
		return mouseWheelLinesDefault
	}
	if mod&uv.ModCtrl != 0 {
		return maxInt(1, w.rows-1)
	}
	if mod&uv.ModShift != 0 {
		return 1
	}
	return mouseWheelLinesDefault
}

func (w *Window) forwardMouseToTerm(event uv.MouseEvent) bool {
	if w == nil || event == nil {
		return false
	}

	w.termMu.Lock()
	term := w.term
	if term != nil {
		term.SendMouse(event)
	}
	w.termMu.Unlock()
	if term == nil {
		return false
	}

	w.markDirty()
	return true
}

func (w *Window) handleMouseWheel(event uv.MouseWheelEvent) bool {
	if w == nil {
		return false
	}

	// Alt-screen: keep existing behaviour. Only forward if the application is
	// actively requesting mouse reporting.
	if w.IsAltScreen() {
		if !w.HasMouseMode() {
			return false
		}
		return w.forwardMouseToTerm(event)
	}

	// Copy mode always consumes the wheel and moves the copy cursor.
	if w.CopyModeActive() {
		step := w.mouseWheelStep(event.Mod)
		switch event.Button {
		case uv.MouseWheelUp:
			w.CopyMove(0, -step)
			return true
		case uv.MouseWheelDown:
			w.CopyMove(0, step)
			return true
		default:
			return false
		}
	}

	// Normal screen: use wheel to scroll terminal history, regardless of mouse reporting.
	if w.ScrollbackLen() <= 0 {
		return false
	}

	step := w.mouseWheelStep(event.Mod)
	switch event.Button {
	case uv.MouseWheelUp:
		w.ScrollUp(step)
		return true
	case uv.MouseWheelDown:
		w.ScrollDown(step)
		return true
	default:
		return false
	}
}

func (w *Window) mouseAbsFromViewport(x, y int) (absX, absY, sbLen int, ok bool) {
	if w == nil {
		return 0, 0, 0, false
	}
	if x < 0 || y < 0 {
		return 0, 0, 0, false
	}

	sbLen = w.ScrollbackLen()

	w.stateMu.Lock()
	offset := w.ScrollbackOffset
	w.stateMu.Unlock()

	topAbsY := sbLen - clampInt(offset, 0, sbLen)
	if topAbsY < 0 {
		topAbsY = 0
	}

	total := sbLen + w.rows
	if total <= 0 {
		return 0, 0, sbLen, false
	}

	absY = clampInt(topAbsY+y, 0, total-1)
	absX = clampInt(x, 0, maxInt(0, w.cols-1))
	return absX, absY, sbLen, true
}

func (w *Window) shouldCaptureMouseForSelection(mod uv.KeyMod) bool {
	if w == nil {
		return false
	}

	// In copy/scrollback views, the user is explicitly interacting with host state, not the app.
	if w.CopyModeActive() || w.ScrollbackModeActive() || w.GetScrollbackOffset() > 0 {
		return true
	}

	// Don't start selection in alt-screen (copy mode isn't supported there).
	if w.IsAltScreen() {
		return false
	}

	// If the app enabled mouse reporting, keep current behaviour unless Shift is held.
	if w.HasMouseMode() && mod&uv.ModShift == 0 {
		return false
	}

	return true
}

func (w *Window) handleMouseSelectClick(event uv.MouseClickEvent) bool {
	if w == nil {
		return false
	}
	if event.Button != uv.MouseLeft {
		return false
	}
	if !w.shouldCaptureMouseForSelection(event.Mod) {
		return false
	}

	// Ensure copy mode is active so selection can be highlighted.
	wasCopyActive := w.CopyModeActive()
	if !wasCopyActive {
		w.EnterCopyMode()
	}
	if !w.CopyModeActive() {
		return false
	}

	absX, absY, sbLen, ok := w.mouseAbsFromViewport(event.X, event.Y)
	if !ok {
		return false
	}
	absX = w.adjustToNonContinuation(absY, absX, -1)

	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return false
	}

	cm.CursorX = absX
	cm.CursorAbsY = absY
	cm.Selecting = true
	cm.SelStartX, cm.SelStartAbsY = absX, absY
	cm.SelEndX, cm.SelEndAbsY = absX, absY
	w.mouseSelectActive = true
	w.mouseSelectStartedCopy = !wasCopyActive
	w.mouseSelectMoved = false
	w.mouseSelection = true

	w.ensureCopyCursorVisibleLocked(sbLen)
	w.stateMu.Unlock()

	w.markDirty()
	return true
}

func (w *Window) handleMouseSelectMotion(event uv.MouseMotionEvent) bool {
	if w == nil {
		return false
	}
	if event.Button != uv.MouseLeft {
		w.stateMu.Lock()
		active := w.mouseSelectActive
		w.stateMu.Unlock()
		if !active {
			return false
		}
	}
	if !w.CopyModeActive() {
		return false
	}

	absX, absY, _, ok := w.mouseAbsFromViewport(event.X, event.Y)
	if !ok {
		w.stateMu.Lock()
		active := w.mouseSelectActive
		w.stateMu.Unlock()
		return active
	}

	w.stateMu.Lock()
	cm := w.CopyMode
	if cm == nil || !cm.Active {
		w.stateMu.Unlock()
		return false
	}
	curX := cm.CursorX
	curY := cm.CursorAbsY
	if !cm.Selecting {
		cm.Selecting = true
		cm.SelStartX, cm.SelStartAbsY = curX, curY
		cm.SelEndX, cm.SelEndAbsY = curX, curY
	}
	w.stateMu.Unlock()

	dx := absX - curX
	dy := absY - curY
	if dx == 0 && dy == 0 {
		return true
	}

	w.CopyMove(dx, dy)
	w.stateMu.Lock()
	if w.mouseSelectActive {
		w.mouseSelectMoved = true
	}
	w.stateMu.Unlock()
	return true
}

func (w *Window) handleMouseSelectRelease(event uv.MouseReleaseEvent) bool {
	if w == nil {
		return false
	}
	if event.Button != uv.MouseLeft {
		w.stateMu.Lock()
		active := w.mouseSelectActive
		w.stateMu.Unlock()
		if !active {
			return false
		}
	}
	if !w.CopyModeActive() {
		w.stateMu.Lock()
		w.mouseSelectActive = false
		w.mouseSelectStartedCopy = false
		w.mouseSelectMoved = false
		w.stateMu.Unlock()
		return true
	}

	absX, absY, _, ok := w.mouseAbsFromViewport(event.X, event.Y)
	moved := false
	if ok {
		w.stateMu.Lock()
		cm := w.CopyMode
		if cm == nil || !cm.Active {
			w.stateMu.Unlock()
			return false
		}
		curX := cm.CursorX
		curY := cm.CursorAbsY
		w.stateMu.Unlock()

		dx := absX - curX
		dy := absY - curY
		if dx != 0 || dy != 0 {
			moved = true
			w.CopyMove(dx, dy)
		}
	}

	w.stateMu.Lock()
	if moved {
		w.mouseSelectMoved = true
	}
	startedCopy := w.mouseSelectStartedCopy
	movedFinal := w.mouseSelectMoved
	w.mouseSelectActive = false
	w.mouseSelectStartedCopy = false
	w.mouseSelectMoved = false
	w.stateMu.Unlock()

	if movedFinal && w.copyMouseSelectionToClipboard() {
		w.notifyToast(mouseSelectionCopiedToast)
	}
	if startedCopy && !movedFinal {
		w.ExitCopyMode()
	}
	return true
}

func (w *Window) copyMouseSelectionToClipboard() bool {
	if w == nil || !w.CopySelectionFromMouseActive() {
		return false
	}
	text := w.CopyYankText()
	if text == "" {
		return false
	}
	if err := writeClipboard(text); err != nil {
		return false
	}
	return true
}

func (w *Window) updateMouseMode(mode ansi.Mode, enabled bool) {
	dec, ok := mode.(ansi.DECMode)
	if !ok {
		return
	}
	var mask uint32
	switch dec {
	case ansi.ModeMouseX10:
		mask = mouseModeX10
	case ansi.ModeMouseNormal:
		mask = mouseModeNormal
	case ansi.ModeMouseHighlight:
		mask = mouseModeHighlight
	case ansi.ModeMouseButtonEvent:
		mask = mouseModeButtonEvent
	case ansi.ModeMouseAnyEvent:
		mask = mouseModeAnyEvent
	case ansi.ModeMouseExtSgr:
		mask = mouseModeSGR
	default:
		return
	}
	for {
		current := w.mouseMode.Load()
		next := current
		if enabled {
			next |= mask
		} else {
			next &^= mask
		}
		if current == next {
			return
		}
		if w.mouseMode.CompareAndSwap(current, next) {
			return
		}
	}
}

func (w *Window) HasMouseMode() bool {
	if w == nil {
		return false
	}
	modes := w.mouseMode.Load()
	return modes&(mouseModeX10|mouseModeNormal|mouseModeHighlight|mouseModeButtonEvent|mouseModeAnyEvent) != 0
}

func (w *Window) AllowsMouseMotion() bool {
	if w == nil {
		return false
	}
	modes := w.mouseMode.Load()
	return modes&(mouseModeButtonEvent|mouseModeAnyEvent) != 0
}

func (w *Window) SendMouse(event uv.MouseEvent) bool {
	if w == nil || event == nil {
		return false
	}
	if w.closed.Load() {
		return false
	}

	if wheel, ok := event.(uv.MouseWheelEvent); ok {
		return w.handleMouseWheel(wheel)
	}

	// In host-controlled views, don't forward mouse to the app.
	if w.CopyModeActive() || w.ScrollbackModeActive() || w.GetScrollbackOffset() > 0 {
		switch ev := event.(type) {
		case uv.MouseClickEvent:
			_ = w.handleMouseSelectClick(ev)
			return true
		case uv.MouseMotionEvent:
			_ = w.handleMouseSelectMotion(ev)
			return true
		case uv.MouseReleaseEvent:
			_ = w.handleMouseSelectRelease(ev)
			return true
		default:
			return true
		}
	}

	// Host-level drag selection when the app is not capturing mouse (or Shift is held).
	switch ev := event.(type) {
	case uv.MouseClickEvent:
		if w.handleMouseSelectClick(ev) {
			return true
		}
	case uv.MouseMotionEvent:
		if w.handleMouseSelectMotion(ev) {
			return true
		}
	case uv.MouseReleaseEvent:
		if w.handleMouseSelectRelease(ev) {
			return true
		}
	}

	if !w.HasMouseMode() {
		return false
	}
	if _, isMotion := event.(uv.MouseMotionEvent); isMotion && !w.AllowsMouseMotion() {
		return false
	}

	return w.forwardMouseToTerm(event)
}
