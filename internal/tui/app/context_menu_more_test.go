package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleContextMenuKeyEscCloses(t *testing.T) {
	m := newTestModelLite()
	m.contextMenu = contextMenuState{
		open:  true,
		items: []contextMenuItem{{ID: contextMenuClose, Label: "Close", Enabled: true}},
		index: 0,
	}
	_, handled := m.handleContextMenuKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.contextMenu.open {
		t.Fatalf("expected menu closed")
	}
}

func TestHandleContextMenuMouseMotionSelectsItem(t *testing.T) {
	m := newTestModelLite()
	m.contextMenu = contextMenuState{
		open:  true,
		x:     10,
		y:     10,
		items: []contextMenuItem{{ID: "a", Label: "A", Enabled: true}, {ID: "b", Label: "B", Enabled: true}},
		index: 0,
	}
	rect, _, _, ok := m.contextMenuLayout()
	if !ok {
		t.Fatalf("expected contextMenuLayout ok")
	}
	msg := tea.MouseMsg{
		Action: tea.MouseActionMotion,
		X:      rect.X,
		Y:      rect.Y + 1,
	}
	_, handled := m.handleContextMenuMouse(msg)
	if !handled {
		t.Fatalf("expected handled")
	}
	if m.contextMenu.index != 1 {
		t.Fatalf("index=%d want=1", m.contextMenu.index)
	}
}

func TestContextMenuMoveSkipsDisabledItems(t *testing.T) {
	m := newTestModelLite()
	m.contextMenu = contextMenuState{
		open: true,
		items: []contextMenuItem{
			{ID: "a", Label: "A", Enabled: true},
			{ID: "b", Label: "B", Enabled: false},
			{ID: "c", Label: "C", Enabled: true},
		},
		index: 0,
	}
	m.contextMenuMove(1)
	if m.contextMenu.index != 2 {
		t.Fatalf("index=%d want=2", m.contextMenu.index)
	}
	m.contextMenuMove(-1)
	if m.contextMenu.index != 0 {
		t.Fatalf("index=%d want=0", m.contextMenu.index)
	}
}

func TestApplyContextMenuDisabledDoesNotClose(t *testing.T) {
	m := newTestModelLite()
	m.contextMenu = contextMenuState{
		open:  true,
		items: []contextMenuItem{{ID: contextMenuClose, Label: "Close", Enabled: false}},
		index: 0,
	}
	if cmd := m.applyContextMenu(); cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	if !m.contextMenu.open {
		t.Fatalf("expected menu still open")
	}
}

func TestApplyContextMenuClosesMenuForKnownActions(t *testing.T) {
	cases := []struct {
		name string
		id   string
	}{
		{name: "add_last", id: contextMenuAddLast},
		{name: "split_right", id: contextMenuSplitRight},
		{name: "split_down", id: contextMenuSplitDown},
		{name: "close", id: contextMenuClose},
		{name: "zoom", id: contextMenuZoom},
		{name: "reset", id: contextMenuReset},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModelLite()
			m.contextMenu = contextMenuState{
				open:    true,
				session: "alpha-1",
				paneID:  "p1",
				items:   []contextMenuItem{{ID: tc.id, Label: "x", Enabled: true}},
				index:   0,
			}
			_ = m.applyContextMenu()
			if m.contextMenu.open {
				t.Fatalf("expected menu closed")
			}
		})
	}
}
