package app

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

func TestBuildDashboardDataOrderStable(t *testing.T) {
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{
			{Name: "Beta", Session: "beta-1", Path: "/beta"},
			{Name: "Alpha", Session: "alpha-1", Path: "/alpha"},
		},
	}
	settings := DashboardConfig{ShowThumbnails: false, PreviewLines: 1}
	sessionsA := []native.SessionSnapshot{
		{Name: "alpha-1", Path: "/alpha", Panes: []native.PaneSnapshot{{ID: "p1", Index: "1", Active: true}}},
		{Name: "beta-1", Path: "/beta", Panes: []native.PaneSnapshot{{ID: "p2", Index: "1", Active: true}}},
		{Name: "gamma-1", Path: "/gamma", Panes: []native.PaneSnapshot{{ID: "p3", Index: "1", Active: true}}},
	}
	sessionsB := []native.SessionSnapshot{sessionsA[2], sessionsA[0], sessionsA[1]}

	resA := buildDashboardData(dashboardSnapshotInput{
		Tab:      TabProject,
		Config:   cfg,
		Settings: settings,
		Sessions: sessionsA,
	})
	resB := buildDashboardData(dashboardSnapshotInput{
		Tab:      TabProject,
		Config:   cfg,
		Settings: settings,
		Sessions: sessionsB,
	})

	for _, project := range resA.Data.Projects {
		if project.ID == "" {
			t.Fatalf("expected project ID set: %#v", project)
		}
	}

	if !reflect.DeepEqual(projectIDs(resA.Data.Projects), projectIDs(resB.Data.Projects)) {
		t.Fatalf("project IDs not stable: %v vs %v", projectIDs(resA.Data.Projects), projectIDs(resB.Data.Projects))
	}
	if !reflect.DeepEqual(projectNames(resA.Data.Projects), projectNames(resB.Data.Projects)) {
		t.Fatalf("project names not stable: %v vs %v", projectNames(resA.Data.Projects), projectNames(resB.Data.Projects))
	}
}

func TestBuildDashboardDataCoalescesConfigByKey(t *testing.T) {
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{
			{Name: "Alpha", Session: "alpha-1", Path: "/alpha"},
			{Name: "Alpha", Session: "alpha-2", Path: "/alpha"},
		},
	}
	settings := DashboardConfig{}
	result := buildDashboardData(dashboardSnapshotInput{
		Tab:      TabProject,
		Config:   cfg,
		Settings: settings,
	})

	if len(result.Data.Projects) != 1 {
		t.Fatalf("projects=%d want 1", len(result.Data.Projects))
	}
	if len(result.Data.Projects[0].Sessions) != 2 {
		t.Fatalf("sessions=%d want 2", len(result.Data.Projects[0].Sessions))
	}
	if result.Data.Projects[0].ID != projectKey("/alpha", "Alpha") {
		t.Fatalf("project ID = %q", result.Data.Projects[0].ID)
	}
}

func TestBuildDashboardDataCoalescesRelativeConfigPath(t *testing.T) {
	absPath, err := filepath.Abs("proj")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{
			{Name: "Proj", Session: "proj-1", Path: "proj"},
		},
	}
	settings := DashboardConfig{ShowThumbnails: false, PreviewLines: 1}
	sessions := []native.SessionSnapshot{
		{Name: "proj-1", Path: absPath, Panes: []native.PaneSnapshot{{ID: "p1", Index: "1", Active: true}}},
	}

	result := buildDashboardData(dashboardSnapshotInput{
		Tab:      TabProject,
		Config:   cfg,
		Settings: settings,
		Sessions: sessions,
	})

	if len(result.Data.Projects) != 1 {
		t.Fatalf("projects=%d want 1", len(result.Data.Projects))
	}
	if got := result.Data.Projects[0].ID; got != projectKey(absPath, "Proj") {
		t.Fatalf("project ID = %q", got)
	}
	if got := result.Data.Projects[0].Path; got != absPath {
		t.Fatalf("project path = %q", got)
	}
}

func projectIDs(projects []ProjectGroup) []string {
	ids := make([]string, 0, len(projects))
	for _, project := range projects {
		ids = append(ids, project.ID)
	}
	return ids
}

func projectNames(projects []ProjectGroup) []string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names
}
