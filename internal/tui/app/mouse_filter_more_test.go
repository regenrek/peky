package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseMotionFilterThrottleAndReset(t *testing.T) {
	m := newTestModelLite()
	m.resize.drag.active = true

	f := NewMouseMotionFilter()
	f.throttle = 0

	msg := tea.MouseMsg{Action: tea.MouseActionMotion, X: 10, Y: 10}
	if got := f.Filter(m, msg); got == nil {
		t.Fatalf("expected msg forwarded")
	}
	if got := f.Filter(m, msg); got != nil {
		t.Fatalf("expected duplicate suppressed")
	}

	m.resize.drag.active = false
	m.state = StateHelp
	if got := f.Filter(m, tea.MouseMsg{Action: tea.MouseActionMotion, X: 11, Y: 11}); got != nil {
		t.Fatalf("expected blocked motion to be dropped")
	}
	if f.lastX != -1 || f.lastY != -1 {
		t.Fatalf("expected reset coordinates")
	}
}
