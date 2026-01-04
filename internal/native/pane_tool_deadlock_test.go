package native

import (
	"testing"
	"time"
)

func TestManagerSetPaneToolDoesNotDeadlock(t *testing.T) {
	m := newTestManager(t)

	pane := &Pane{ID: "p-1"}
	m.mu.Lock()
	if m.panes == nil {
		m.panes = make(map[string]*Pane)
	}
	m.panes[pane.ID] = pane
	m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		_ = m.SetPaneTool(pane.ID, "codex")
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(1 * time.Second):
		t.Fatalf("SetPaneTool did not return; possible Manager.mu self-deadlock")
	}
}
