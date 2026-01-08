package app

import "testing"

func TestSelectionForProjectIDUsesRememberedSelection(t *testing.T) {
	m := newTestModelLite()
	got := m.selectionForProjectID("proj")
	if got.ProjectID != "proj" || got.Session != "" || got.Pane != "" {
		t.Fatalf("got=%#v", got)
	}
	m.rememberSelection(selectionState{ProjectID: "proj", Session: "sess", Pane: "1"})
	got = m.selectionForProjectID("proj")
	if got.ProjectID != "proj" || got.Session != "sess" || got.Pane != "1" {
		t.Fatalf("got=%#v", got)
	}
}
