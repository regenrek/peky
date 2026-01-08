package transform

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/workspace"
)

func TestParseIndex(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"  ", 0},
		{"01", 1},
		{"12", 12},
		{"x", 0},
		{"1x", 0},
	}
	for _, tc := range cases {
		if got := parseIndex(tc.in); got != tc.want {
			t.Fatalf("parseIndex(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestGroupNameFromPath(t *testing.T) {
	if got := groupNameFromPath("", "fallback"); got != "fallback" {
		t.Fatalf("empty path=%q", got)
	}
	if got := groupNameFromPath(" / ", "fallback"); got != "fallback" {
		t.Fatalf("root path=%q", got)
	}
	if got := groupNameFromPath("/tmp/myproj", "fallback"); got != "myproj" {
		t.Fatalf("basename=%q", got)
	}
}

func TestResolveSelection(t *testing.T) {
	projects := []output.ProjectSnapshot{
		{ID: "p1", Name: "P1", Sessions: []output.SessionSnapshot{{Name: "s1"}}},
		{ID: "p2", Name: "P2", Sessions: []output.SessionSnapshot{{Name: "s2"}}},
	}
	pid, sid := resolveSelection(projects, "s2")
	if pid != "p2" || sid != "s2" {
		t.Fatalf("selection=%q/%q", pid, sid)
	}
}

func TestBuildSnapshotAddsUnknownProject(t *testing.T) {
	now := time.Now().UTC()
	sessions := []native.SessionSnapshot{{
		Name:       "s1",
		Path:       "/tmp/unknown",
		LayoutName: "auto",
		CreatedAt:  now,
		Panes: []native.PaneSnapshot{{
			ID:         "p-1",
			Index:      "1",
			LastActive: now,
		}},
	}}
	ws := workspace.Workspace{Projects: []workspace.Project{}}
	snap := BuildSnapshot(sessions, ws, "", "")
	if len(snap.Projects) != 1 {
		t.Fatalf("projects=%d", len(snap.Projects))
	}
	if snap.Projects[0].ID == "" || len(snap.Projects[0].Sessions) != 1 {
		t.Fatalf("project=%#v", snap.Projects[0])
	}
}
