package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOfflineScrollToggleAndPaneView(t *testing.T) {
	m := newTestModelLite()
	project := &m.data.Projects[0]
	session := &project.Sessions[0]
	pane := &session.Panes[0]
	pane.Disconnected = true
	pane.Preview = []string{"L0", "L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9"}

	if _, handled := m.handleOfflineScrollInput(keyMsgFromTea(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})); !handled {
		t.Fatalf("expected handled toggle")
	}
	if !m.offlineScrollActiveFor(pane.ID) {
		t.Fatalf("expected offline scroll active")
	}
	if got := m.offlineScrollOffset(pane.ID); got != 0 {
		t.Fatalf("offset=%d want=0", got)
	}

	view := m.offlinePaneView(pane, 20, 3)
	if view != "L7\nL8\nL9" {
		t.Fatalf("view=%q", view)
	}

	if _, handled := m.handleOfflineScrollInput(keyMsgFromTea(tea.KeyMsg{Type: tea.KeyUp})); !handled {
		t.Fatalf("expected handled up")
	}
	if got := m.offlineScrollOffset(pane.ID); got != 1 {
		t.Fatalf("offset=%d want=1", got)
	}
	if _, handled := m.handleOfflineScrollInput(keyMsgFromTea(tea.KeyMsg{Type: tea.KeyPgUp})); !handled {
		t.Fatalf("expected handled pgup")
	}
	if got := m.offlineScrollOffset(pane.ID); got != 4 {
		t.Fatalf("offset=%d want=4", got)
	}

	m.setOfflineScrollOffset(*pane, 2)
	view = m.offlinePaneView(pane, 20, 3)
	if view != "L5\nL6\nL7" {
		t.Fatalf("view=%q", view)
	}

	if _, handled := m.handleOfflineScrollInput(keyMsgFromTea(tea.KeyMsg{Type: tea.KeyEsc})); !handled {
		t.Fatalf("expected handled esc")
	}
	if m.offlineScrollActiveFor(pane.ID) {
		t.Fatalf("expected offline scroll cleared")
	}
}

func TestPruneOfflineScrollDropsNonDisconnectedPanes(t *testing.T) {
	m := newTestModelLite()
	pane := &m.data.Projects[0].Sessions[0].Panes[0]
	pane.Disconnected = true
	pane.Preview = []string{"a", "b", "c"}

	m.ensureOfflineScrollMap()
	m.offlineScroll[pane.ID] = 1
	m.offlineScrollViewport[pane.ID] = 2
	m.offlineScroll["gone"] = 5
	m.offlineScrollViewport["gone"] = 1

	m.pruneOfflineScroll()
	if _, ok := m.offlineScroll["gone"]; ok {
		t.Fatalf("expected pruned offset")
	}
	if _, ok := m.offlineScrollViewport["gone"]; ok {
		t.Fatalf("expected pruned viewport")
	}
	if _, ok := m.offlineScroll[pane.ID]; !ok {
		t.Fatalf("expected keep disconnected pane offset")
	}
}
