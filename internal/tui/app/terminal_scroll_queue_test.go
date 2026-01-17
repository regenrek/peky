package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestWheelRoutesToTerminalScrollWhenNotInMouseMode(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}

	hit := mouse.PaneHit{
		PaneID: "p1",
		Selection: mouse.Selection{
			ProjectID: projectKey("/alpha", "Alpha"),
			Session:   "alpha-1",
			Pane:      "1",
		},
		Content: mouse.Rect{X: 0, Y: 0, W: 10, H: 5},
	}

	cmd := m.forwardMouseEvent(hit, tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if got := len(m.mouseSendQueue); got != 0 {
		t.Fatalf("mouse send queue len=%d want 0", got)
	}
	if got := len(m.terminalScrollQueue); got != 1 {
		t.Fatalf("terminal scroll queue len=%d want 1", got)
	}
	item := m.terminalScrollQueue[0]
	if item.action != sessiond.TerminalScrollUp {
		t.Fatalf("action=%v want %v", item.action, sessiond.TerminalScrollUp)
	}
	if item.lines != 3 {
		t.Fatalf("lines=%d want 3", item.lines)
	}
}

func TestWheelRoutesToSendMouseWhenPaneHasMouse(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}
	m.hardRaw = true
	m.paneHasMouse = map[string]bool{"p1": true}

	hit := mouse.PaneHit{
		PaneID: "p1",
		Selection: mouse.Selection{
			ProjectID: projectKey("/alpha", "Alpha"),
			Session:   "alpha-1",
			Pane:      "1",
		},
		Content: mouse.Rect{X: 0, Y: 0, W: 10, H: 5},
	}

	cmd := m.forwardMouseEvent(hit, tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if got := len(m.terminalScrollQueue); got != 0 {
		t.Fatalf("terminal scroll queue len=%d want 0", got)
	}
	if got := len(m.mouseSendQueue); got != 1 {
		t.Fatalf("mouse send queue len=%d want 1", got)
	}
	if !m.mouseSendQueue[0].payload.Wheel {
		t.Fatalf("expected wheel payload")
	}
}

func TestWheelCoalescesTerminalScrollLines(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}

	hit := mouse.PaneHit{
		PaneID: "p1",
		Selection: mouse.Selection{
			ProjectID: projectKey("/alpha", "Alpha"),
			Session:   "alpha-1",
			Pane:      "1",
		},
		Content: mouse.Rect{X: 0, Y: 0, W: 10, H: 5},
	}

	_ = m.forwardMouseEvent(hit, tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	_ = m.forwardMouseEvent(hit, tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})

	if got := len(m.terminalScrollQueue); got != 1 {
		t.Fatalf("terminal scroll queue len=%d want 1", got)
	}
	if got := m.terminalScrollQueue[0].lines; got != 6 {
		t.Fatalf("lines=%d want 6", got)
	}
}
