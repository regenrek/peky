package app

import "testing"

func TestSelectionFromFocusPaneID(t *testing.T) {
	groups := []ProjectGroup{{
		ID:   "proj",
		Name: "Proj",
		Sessions: []SessionItem{{
			Name: "sess",
			Panes: []PaneItem{
				{ID: "p1", Index: "1"},
				{ID: "p2", Index: "2"},
			},
		}},
	}}

	sel := selectionFromFocus(buildFocusIndex(groups), "", "p2")
	if sel.ProjectID != "proj" || sel.Session != "sess" || sel.Pane != "2" {
		t.Fatalf("selectionFromFocus() = %#v", sel)
	}
}

func TestSelectionFromFocusSession(t *testing.T) {
	groups := []ProjectGroup{{
		ID:   "proj",
		Name: "Proj",
		Sessions: []SessionItem{{
			Name: "sess",
			Panes: []PaneItem{
				{ID: "p1", Index: "1"},
				{ID: "p2", Index: "2", Active: true},
			},
		}},
	}}

	sel := selectionFromFocus(buildFocusIndex(groups), "sess", "missing")
	if sel.ProjectID != "proj" || sel.Session != "sess" || sel.Pane != "2" {
		t.Fatalf("selectionFromFocus(session) = %#v", sel)
	}
}
