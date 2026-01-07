package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestContextMenuLayoutClamps(t *testing.T) {
	m := newTestModelLite()
	hit := mouse.PaneHit{
		PaneID: "p1",
		Selection: mouse.Selection{
			ProjectID: projectKey("/alpha", "Alpha"),
			Session:   "alpha-1",
			Pane:      "1",
		},
	}
	m.openContextMenu(999, 999, hit)
	rect, _, _, ok := m.contextMenuLayout()
	if !ok {
		t.Fatalf("expected context menu layout")
	}
	layout, ok := m.dashboardLayoutInternal("contextMenuTest")
	if !ok {
		t.Fatalf("expected dashboard layout")
	}
	maxX := layout.padLeft + layout.contentWidth
	maxY := layout.padTop + layout.contentHeight
	if rect.X+rect.W > maxX {
		t.Fatalf("menu overflows width: rect=%+v maxX=%d", rect, maxX)
	}
	if rect.Y+rect.H > maxY {
		t.Fatalf("menu overflows height: rect=%+v maxY=%d", rect, maxY)
	}
}

func TestContextMenuHoverUpdatesSelection(t *testing.T) {
	m := newTestModelLite()
	hit := mouse.PaneHit{
		PaneID: "p1",
		Selection: mouse.Selection{
			ProjectID: projectKey("/alpha", "Alpha"),
			Session:   "alpha-1",
			Pane:      "1",
		},
	}
	m.openContextMenu(10, 10, hit)
	if !m.contextMenu.open {
		t.Fatalf("expected context menu open")
	}
	rect, _, _, ok := m.contextMenuLayout()
	if !ok {
		t.Fatalf("expected context menu rect")
	}
	if rect.H < 3 {
		t.Fatalf("expected at least 3 menu items")
	}
	start := m.contextMenu.index
	msg := tea.MouseMsg{
		X:      rect.X + 1,
		Y:      rect.Y + 2,
		Action: tea.MouseActionMotion,
		Button: tea.MouseButtonNone,
	}
	_, handled := m.handleContextMenuMouse(msg)
	if !handled {
		t.Fatalf("expected motion handled")
	}
	if m.contextMenu.index == start {
		t.Fatalf("expected selection to change on hover")
	}
	if m.contextMenu.index != 2 {
		t.Fatalf("selection=%d want %d", m.contextMenu.index, 2)
	}
}
