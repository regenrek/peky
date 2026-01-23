package app

import "testing"

func TestMarkPaneInputDisabled(t *testing.T) {
	m := &Model{}
	m.markPaneInputDisabled("p1")
	if !m.isPaneInputDisabled("p1") {
		t.Fatalf("expected pane input disabled")
	}
}

func TestSelectionForPaneID(t *testing.T) {
	m := newTestModelLite()
	selection, ok := m.selectionForPaneID("p2")
	if !ok {
		t.Fatalf("expected selection for pane")
	}
	if selection.ProjectID == "" || selection.Session == "" || selection.Pane == "" {
		t.Fatalf("unexpected selection: %#v", selection)
	}
	if _, ok := m.selectionForPaneID("missing"); ok {
		t.Fatalf("expected missing selection")
	}
}

func TestReconcilePaneInputDisabled(t *testing.T) {
	m := newTestModelLite()
	m.paneInputDisabled = map[string]struct{}{
		"p1": {},
		"p2": {},
	}
	pane := m.paneByID("p1")
	if pane == nil {
		t.Fatalf("expected pane p1")
	}
	pane.Dead = true

	m.reconcilePaneInputDisabled()
	if !m.isPaneInputDisabled("p1") {
		t.Fatalf("expected dead pane disabled")
	}
	if m.isPaneInputDisabled("p2") {
		t.Fatalf("expected live pane removed")
	}
}
