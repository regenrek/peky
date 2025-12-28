package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectionTabNavigation(t *testing.T) {
	m := newTestModelLite()

	m.tab = TabDashboard
	m.selection = selectionState{Session: "alpha-1"}
	m.selectTab(1)
	if m.tab != TabProject || m.selection.Project != "Alpha" {
		t.Fatalf("expected project tab Alpha, got tab=%v sel=%#v", m.tab, m.selection)
	}
	m.selectTab(1)
	if m.selection.Project != "Beta" {
		t.Fatalf("expected project Beta, got %#v", m.selection)
	}
	m.selectTab(1)
	if m.tab != TabDashboard {
		t.Fatalf("expected dashboard tab")
	}
}

func TestSelectionSessionAndPaneNavigation(t *testing.T) {
	m := newTestModelLite()

	m.tab = TabProject
	m.selection = selectionState{Project: "Alpha", Session: "alpha-1"}
	m.selectSession(1)
	if m.selection.Session != "alpha-2" {
		t.Fatalf("expected session alpha-2, got %#v", m.selection)
	}

	m.selection = selectionState{Project: "Alpha", Session: "alpha-1"}
	m.selectSessionOrPane(1)
	if m.selection.Pane != "1" {
		t.Fatalf("expected pane 1, got %#v", m.selection)
	}
	m.selectSessionOrPane(1)
	if m.selection.Pane != "2" {
		t.Fatalf("expected pane 2, got %#v", m.selection)
	}

	m.selection = selectionState{Project: "Alpha", Session: "alpha-1"}
	m.selectPane(1)
	if m.selection.Pane == "" {
		t.Fatalf("expected pane selected")
	}
	prevVersion := m.selectionVersion
	m.cyclePane(1)
	if m.selectionVersion <= prevVersion {
		t.Fatalf("expected selection version increment")
	}
}

func TestSelectionDashboardAndToggle(t *testing.T) {
	m := newTestModelLite()

	m.tab = TabDashboard
	m.selection = selectionState{Session: "alpha-1"}
	m.selectDashboardPane(1)
	if m.selection.Project == "" || m.selection.Session == "" || m.selection.Pane == "" {
		t.Fatalf("expected dashboard pane selection, got %#v", m.selection)
	}

	m.selectDashboardProject(1)
	if m.selection.Project == "" {
		t.Fatalf("expected dashboard project selection")
	}

	m.togglePanes()
	if !m.expandedSessions["alpha-1"] {
		t.Fatalf("expected pane toggle to flip state")
	}

	if m.selectedProject() == nil || m.selectedSession() == nil || m.selectedPane() == nil {
		t.Fatalf("expected selected project/session/pane")
	}
}

func TestSelectionHelpersFiltering(t *testing.T) {
	m := newTestModelLite()
	m.filterInput.SetValue("beta")

	if got := m.filteredSessions(m.data.Projects[0].Sessions); len(got) != 0 {
		t.Fatalf("expected alpha sessions filtered out, got %#v", got)
	}

	columns := collectDashboardColumns(m.data.Projects)
	filtered := m.filteredDashboardColumns(columns)
	if len(filtered) != 2 || len(filtered[1].Panes) == 0 {
		t.Fatalf("expected filtered dashboard columns, got %#v", filtered)
	}
}

func TestRefreshSelectionForProjectConfig(t *testing.T) {
	m := newTestModelLite()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".peakypanes.yml")
	if err := os.WriteFile(cfgPath, []byte("session: alpha"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	m.data.Projects[0].Path = dir
	m.selection = selectionState{Project: "Alpha"}
	projectPath := normalizeProjectPath(dir)
	m.projectConfigState = map[string]projectConfigState{
		projectPath: {exists: false},
	}

	if !m.refreshSelectionForProjectConfig() {
		t.Fatalf("expected selection refresh")
	}
	if m.selection.Project == "" {
		t.Fatalf("expected selection populated")
	}
}

func TestResolveSelections(t *testing.T) {
	projects := sampleProjects()

	sel := resolveSelection(projects, selectionState{Project: "missing"})
	if sel.Project != "Alpha" || sel.Session == "" {
		t.Fatalf("expected default selection, got %#v", sel)
	}

	dashSel := resolveDashboardSelection(projects, selectionState{Session: "beta-1"})
	if dashSel.Project != "Beta" || dashSel.Session != "beta-1" {
		t.Fatalf("expected dashboard selection, got %#v", dashSel)
	}

	columns := collectDashboardColumns(projects)
	byProject := resolveDashboardSelectionFromColumns(columns, selectionState{Project: "Alpha"})
	if byProject.Project != "Alpha" {
		t.Fatalf("expected project Alpha, got %#v", byProject)
	}

	panes := projects[0].Sessions[0].Panes
	if got := resolvePaneSelection("2", panes); got != "2" {
		t.Fatalf("expected pane 2, got %q", got)
	}
	if got := resolvePaneSelection("missing", panes); got == "" {
		t.Fatalf("expected fallback pane, got %q", got)
	}

	if findProject(projects, "Beta") == nil || findSession(&projects[0], "alpha-1") == nil {
		t.Fatalf("expected project/session lookup")
	}
	if _, sess := findProjectForSession(projects, "alpha-1"); sess == nil {
		t.Fatalf("expected session lookup by name")
	}
}
