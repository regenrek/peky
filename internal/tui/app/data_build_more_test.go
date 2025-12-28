package app

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

func TestDashboardGroupIndexMerge(t *testing.T) {
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{{
			Name:    "Alpha",
			Session: "alpha-1",
			Path:    "/alpha",
			Layout:  "dev-3",
		}},
	}
	settings := DashboardConfig{
		ShowThumbnails: true,
		PreviewLines:   12,
		ThumbnailLines: 1,
		IdleThreshold:  time.Second,
	}
	idx := newDashboardGroupIndex(-5)
	idx.addConfigProjects(cfg, settings)
	if len(idx.groups) != 1 || idx.groups[0].Name != "Alpha" {
		t.Fatalf("expected config group, got %#v", idx.groups)
	}

	nativeSessions := []native.SessionSnapshot{{
		Name:       "alpha-1",
		Path:       "/alpha",
		LayoutName: "dev-3",
		Panes: []native.PaneSnapshot{{
			ID:     "p1",
			Index:  "1",
			Active: true,
			Width:  10,
			Height: 5,
		}},
	}}
	idx.mergeNativeSessions(nativeSessions, settings)
	if len(idx.groups[0].Sessions) != 1 || idx.groups[0].Sessions[0].Status != StatusRunning {
		t.Fatalf("expected merged running session, got %#v", idx.groups[0].Sessions)
	}
}

func TestApplySessionThumbnails(t *testing.T) {
	groups := []ProjectGroup{{
		Name: "Alpha",
		Sessions: []SessionItem{{
			Name:   "alpha-1",
			Status: StatusRunning,
			Panes: []PaneItem{{
				Index:  "1",
				Active: true,
				Status: PaneStatusRunning,
				Preview: []string{
					"line1",
				},
			}},
		}},
	}}
	settings := DashboardConfig{ShowThumbnails: true, ThumbnailLines: 1}
	applySessionThumbnails(groups, settings)
	if groups[0].Sessions[0].Thumbnail.Line == "" {
		t.Fatalf("expected thumbnail line")
	}

	settings.ShowThumbnails = false
	groups[0].Sessions[0].Thumbnail = PaneSummary{}
	applySessionThumbnails(groups, settings)
	if groups[0].Sessions[0].Thumbnail.Line != "" {
		t.Fatalf("expected no thumbnail when disabled")
	}
}

func TestSessionThumbnailFromDataEmpty(t *testing.T) {
	if got := sessionThumbnailFromData(nil, DashboardConfig{}); got.Line != "" {
		t.Fatalf("expected empty thumbnail")
	}
	if got := sessionThumbnailFromData(&SessionItem{}, DashboardConfig{}); got.Line != "" {
		t.Fatalf("expected empty thumbnail for no panes")
	}
}

func TestBuildDashboardDataOrdersProjects(t *testing.T) {
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{
			{Name: "Beta", Session: "beta-1", Path: "/beta"},
			{Name: "Aidex", Session: "aidex-1", Path: "/aidex"},
		},
	}
	settings := DashboardConfig{PreviewLines: 2, ThumbnailLines: 1}
	sessions := []native.SessionSnapshot{
		{Name: "beta-1", Path: "/beta", Panes: []native.PaneSnapshot{{ID: "p1", Index: "0", Active: true, Width: 10, Height: 5}}},
		{Name: "aidex-1", Path: "/aidex", Panes: []native.PaneSnapshot{{ID: "p2", Index: "0", Active: true, Width: 10, Height: 5}}},
		{Name: "gamma-1", Path: "/gamma", Panes: []native.PaneSnapshot{{ID: "p3", Index: "0", Active: true, Width: 10, Height: 5}}},
		{Name: "alpha-1", Path: "/alpha", Panes: []native.PaneSnapshot{{ID: "p4", Index: "0", Active: true, Width: 10, Height: 5}}},
	}
	result := buildDashboardData(dashboardSnapshotInput{
		Tab:      TabProject,
		Config:   cfg,
		Settings: settings,
		Sessions: sessions,
	})
	got := make([]string, 0, len(result.Data.Projects))
	for _, project := range result.Data.Projects {
		got = append(got, project.Name)
	}
	want := []string{"Beta", "Aidex", "alpha", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("project order length=%d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("project order[%d]=%q want %q (%v)", i, got[i], want[i], got)
		}
	}
}
