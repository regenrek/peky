package app

import (
	"testing"
)

func TestDashboardNavBranches(t *testing.T) {
	m := newTestModelLite()
	m.tab = TabDashboard

	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('k'))); !handled {
		t.Fatalf("expected session up handled on dashboard tab")
	}
	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('j'))); !handled {
		t.Fatalf("expected session down handled on dashboard tab")
	}
	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('K'))); !handled {
		t.Fatalf("expected session only up handled on dashboard tab")
	}
	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('J'))); !handled {
		t.Fatalf("expected session only down handled on dashboard tab")
	}

	if _, handled := m.handlePaneNav(keyMsgFromTea(keyRune('n'))); !handled {
		t.Fatalf("expected pane next handled on dashboard tab")
	}
	if _, handled := m.handlePaneNav(keyMsgFromTea(keyRune('p'))); !handled {
		t.Fatalf("expected pane prev handled on dashboard tab")
	}

	m.tab = TabProject
	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('k'))); !handled {
		t.Fatalf("expected session up handled on project tab")
	}
	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('j'))); !handled {
		t.Fatalf("expected session down handled on project tab")
	}
	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('K'))); !handled {
		t.Fatalf("expected session only up handled on project tab")
	}
	if _, handled := m.handleSessionNav(keyMsgFromTea(keyRune('J'))); !handled {
		t.Fatalf("expected session only down handled on project tab")
	}

	if _, handled := m.handlePaneNav(keyMsgFromTea(keyRune('n'))); !handled {
		t.Fatalf("expected pane next handled on project tab")
	}
	if _, handled := m.handlePaneNav(keyMsgFromTea(keyRune('p'))); !handled {
		t.Fatalf("expected pane prev handled on project tab")
	}
}
