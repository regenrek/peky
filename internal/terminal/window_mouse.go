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

	// If the application enabled mouse reporting, preserve existing behaviour and forward.
	if w.HasMouseMode() {
		return w.forwardMouseToTerm(event)
	}

	// Otherwise, use wheel to scroll terminal history.
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

	if !w.HasMouseMode() {
		return false
	}
	if _, isMotion := event.(uv.MouseMotionEvent); isMotion && !w.AllowsMouseMotion() {
		return false
	}

	return w.forwardMouseToTerm(event)
}
