package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestDashboardSelectionHelpers(t *testing.T) {
	groups := sampleProjects()
	columns := collectDashboardColumns(groups)
	if len(columns) == 0 {
		t.Fatalf("expected columns")
	}

	if got := dashboardSelectedProject(columns, selectionState{Project: "Beta"}); got != "Beta" {
		t.Fatalf("expected Beta, got %q", got)
	}
	if got := dashboardSelectedProject(columns, selectionState{Session: "alpha-2"}); got != "Alpha" {
		t.Fatalf("expected Alpha, got %q", got)
	}

	if idx := dashboardPaneIndex(columns[0].Panes, selectionState{Session: "alpha-1", Pane: "2"}); idx < 0 {
		t.Fatalf("expected pane index for alpha-1 pane 2")
	}

	selected := resolveDashboardSelectionFromColumns(columns, selectionState{Project: "Beta"})
	if selected.Project != "Beta" {
		t.Fatalf("expected selection for Beta, got %#v", selected)
	}

	resolved := resolveSelection(groups, selectionState{Project: "Missing"})
	if resolved.Project == "" || resolved.Session == "" {
		t.Fatalf("expected resolved selection fallback, got %#v", resolved)
	}

	panes := []PaneItem{
		{Index: "1"},
		{Index: "2", Active: true},
	}
	if got := resolvePaneSelection("", panes); got != "2" {
		t.Fatalf("expected active pane selection, got %q", got)
	}
}

func TestGroupForSessionAndHiddenProject(t *testing.T) {
	idx := newDashboardGroupIndex(1)
	idx.groups = []ProjectGroup{{Name: "Alpha", Path: "/alpha"}}
	idx.bySession["alpha-1"] = 0
	idx.byPath["/alpha"] = 0
	if group := idx.groupForSession("alpha-1", "/alpha"); group == nil || group.Name != "Alpha" {
		t.Fatalf("expected group for session")
	}

	settings := DashboardConfig{HiddenProjects: map[string]struct{}{"/alpha": {}}}
	if !isHiddenProject(settings, "/alpha", "Alpha") {
		t.Fatalf("expected hidden project by path")
	}
	settings = DashboardConfig{HiddenProjects: map[string]struct{}{"alpha": {}}}
	if !isHiddenProject(settings, "", "Alpha") {
		t.Fatalf("expected hidden project by name")
	}
}

func TestPaneStatusClassification(t *testing.T) {
	matcher := statusMatcher{
		success: regexp.MustCompile("ok"),
		error:   regexp.MustCompile("err"),
		running: regexp.MustCompile("run"),
	}
	if status, ok := classifyPaneFromLines([]string{"run build"}, matcher); !ok || status != PaneStatusRunning {
		t.Fatalf("expected running from lines")
	}
	if status, ok := classifyPaneFromLines([]string{"err failed"}, matcher); !ok || status != PaneStatusError {
		t.Fatalf("expected error from lines")
	}
	if status, ok := classifyPaneFromLines([]string{"ok done"}, matcher); !ok || status != PaneStatusDone {
		t.Fatalf("expected done from lines")
	}

	pane := PaneItem{Dead: true, DeadStatus: 2}
	if status := classifyPane(pane, nil, DashboardConfig{}, time.Now()); status != PaneStatusError {
		t.Fatalf("expected dead pane error")
	}
	pane = PaneItem{Dead: true, DeadStatus: 0}
	if status := classifyPane(pane, nil, DashboardConfig{}, time.Now()); status != PaneStatusDone {
		t.Fatalf("expected dead pane done")
	}

	now := time.Now()
	pane = PaneItem{LastActive: now.Add(-2 * time.Second)}
	settings := DashboardConfig{IdleThreshold: time.Second}
	if !paneIdle(pane, settings, now) {
		t.Fatalf("expected pane idle")
	}

	summary := paneSummaryLine(PaneItem{Preview: []string{"", "last"}}, 2)
	if summary != "last" {
		t.Fatalf("expected summary from preview, got %q", summary)
	}
	summary = paneSummaryLine(PaneItem{Title: "title"}, 0)
	if summary != "title" {
		t.Fatalf("expected summary from title, got %q", summary)
	}
}

func TestClassifyPaneFromAgentState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PEAKYPANES_AGENT_STATE_DIR", dir)
	state := map[string]any{
		"state":              "running",
		"tool":               "codex",
		"updated_at_unix_ms": time.Now().UnixMilli(),
		"pane_id":            "pane-1",
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pane-1.json"), data, 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}

	settings := DashboardConfig{AgentDetection: AgentDetectionConfig{Codex: true}}
	status, ok := classifyPaneFromAgent(PaneItem{ID: "pane-1"}, []string{"running"}, settings, time.Now())
	if !ok || status != PaneStatusRunning {
		t.Fatalf("expected agent status running, got %v ok=%v", status, ok)
	}
}
