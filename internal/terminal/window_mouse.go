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

func (w *Window) handleMouseWheel(event uv.MouseWheelEvent, route MouseRoute) bool {
	if w == nil {
		return false
	}

	if w.IsAltScreen() {
		return w.handleAltScreenMouseWheel(event, route)
	}

	// Copy/scrollback views: wheel scrolls host viewport. This keeps selections
	// anchored and avoids "selection grows while scrolling" behavior.
	if w.CopyModeActive() || w.ScrollbackModeActive() || w.GetScrollbackOffset() > 0 {
		return w.handleHostWheel(event, true)
	}

	return w.handleHostWheel(event, false)
}

func (w *Window) handleAltScreenMouseWheel(event uv.MouseWheelEvent, route MouseRoute) bool {
	if w == nil {
		return false
	}

	// Alt-screen: in host selection routing, prefer host scrollback to avoid
	// injecting mouse sequences into the application when the user is not
	// explicitly interacting with it.
	if route == MouseRouteHostSelection {
		return w.handleHostWheel(event, false)
	}

	// When routing to the app, keep existing behaviour: only forward if the
	// application is actively requesting mouse reporting.
	if !w.HasMouseMode() {
		// Best-effort: if the app isn't requesting mouse reporting, allow wheel to
		// scroll host scrollback when available.
		return w.handleHostWheel(event, false)
	}
	return w.forwardMouseToTerm(event)
}

func (w *Window) handleHostWheel(event uv.MouseWheelEvent, consumeWhenEmpty bool) bool {
	if w == nil {
		return false
	}
	sbLen := w.ScrollbackLen()
	if sbLen <= 0 {
		return consumeWhenEmpty
	}
	step := w.mouseWheelStep(event.Mod)
	return w.applyWheelScroll(event.Button, step)
}

func (w *Window) applyWheelScroll(button uv.MouseButton, step int) bool {
	if w == nil {
		return false
	}
	switch button {
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

func (w *Window) shouldCaptureMouseForSelection(mod uv.KeyMod, route MouseRoute) bool {
	if w == nil {
		return false
	}

	if route == MouseRouteHostSelection {
		return true
	}
	if route == MouseRouteApp {
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

func (w *Window) SendMouse(event uv.MouseEvent, route MouseRoute) bool {
	if w == nil || event == nil {
		return false
	}
	if w.closed.Load() {
		return false
	}

	if wheel, ok := event.(uv.MouseWheelEvent); ok {
		return w.handleMouseWheel(wheel, route)
	}

	if w.handleMouseInHostModes(event) {
		return true
	}

	if route == MouseRouteHostSelection {
		_ = w.handleMouseSelection(event, route)
		return true
	}
	if route != MouseRouteApp {
		// Host-level drag selection when the app is not capturing mouse (or Shift is held).
		if w.handleMouseSelection(event, route) {
			return true
		}
	}

	if !w.allowMouseForwarding(event) {
		return false
	}

	return w.forwardMouseToTerm(event)
}

func (w *Window) handleMouseInHostModes(event uv.MouseEvent) bool {
	if !w.CopyModeActive() && !w.ScrollbackModeActive() && w.GetScrollbackOffset() == 0 {
		return false
	}
	switch ev := event.(type) {
	case uv.MouseClickEvent:
		_ = w.handleMouseSelectPress(ev, MouseRouteHostSelection)
	case uv.MouseMotionEvent:
		_ = w.handleMouseSelectMotion(ev, MouseRouteHostSelection)
	case uv.MouseReleaseEvent:
		_ = w.handleMouseSelectRelease(ev)
	}
	return true
}

func (w *Window) allowMouseForwarding(event uv.MouseEvent) bool {
	if !w.HasMouseMode() {
		return false
	}
	if _, isMotion := event.(uv.MouseMotionEvent); isMotion && !w.AllowsMouseMotion() {
		return false
	}
	return true
}
