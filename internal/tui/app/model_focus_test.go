package app

import "testing"

func TestFocusSelectionCmdEmptySelectionReturnsNil(t *testing.T) {
	m := newTestModel(t)
	if cmd := m.focusSelectionCmd(selectionState{}); cmd != nil {
		t.Fatalf("expected nil cmd for empty selection")
	}
}

func TestFocusSelectionCmdEmptySessionReturnsNil(t *testing.T) {
	m := newTestModel(t)
	if cmd := m.focusSelectionCmd(selectionState{Pane: "0"}); cmd != nil {
		t.Fatalf("expected nil cmd for empty session")
	}
}
