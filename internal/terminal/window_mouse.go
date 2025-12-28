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
	if !w.HasMouseMode() {
		return false
	}
	if _, isMotion := event.(uv.MouseMotionEvent); isMotion && !w.AllowsMouseMotion() {
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
