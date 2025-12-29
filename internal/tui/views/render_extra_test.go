package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
)

func TestViewDashboardContentRenders(t *testing.T) {
	input := textinput.New()
	input.SetValue("hello")
	m := Model{
		Width:                    80,
		Height:                   24,
		HeaderLine:               "🎩 Peaky Panes",
		EmptyStateMessage:        "empty",
		ShowThumbnails:           true,
		QuickReplyInput:          input,
		DashboardColumns:         []DashboardColumn{{ProjectID: "proj", ProjectName: "Proj", ProjectPath: "", Panes: []DashboardPane{{ProjectID: "proj", ProjectName: "Proj", SessionName: "sess", Pane: Pane{Index: "0"}}}}},
		DashboardSelectedProject: "proj",
		Projects: []Project{{
			Name: "Proj",
			Sessions: []Session{{
				Name:       "sess",
				Status:     sessionRunning,
				PaneCount:  1,
				ActivePane: "0",
				Panes: []Pane{{
					Index:  "0",
					Title:  "shell",
					Active: true,
					Width:  80,
					Height: 24,
				}},
			}},
		}},
		SidebarProject:  &Project{Name: "Proj"},
		SidebarSessions: []Session{{Name: "sess", Status: sessionRunning}},
	}

	out := m.viewDashboardContent()
	if !strings.Contains(out, "Peaky Panes") {
		t.Fatalf("viewDashboardContent() missing header")
	}
	if !strings.Contains(out, "Quick Reply") {
		t.Fatalf("viewDashboardContent() missing quick reply")
	}
}

func TestRenderPanePreviewModes(t *testing.T) {
	panes := []Pane{
		{Index: "0", Title: "main", Left: 0, Top: 0, Width: 50, Height: 20, Active: true, Status: paneStatusRunning},
		{Index: "1", Title: "side", Left: 50, Top: 0, Width: 50, Height: 20, Active: false, Status: paneStatusIdle},
	}
	grid := renderPanePreview(panes, 40, 10, "grid", true, "0", false)
	if strings.TrimSpace(grid) == "" {
		t.Fatalf("renderPanePreview(grid) empty")
	}
	layout := renderPanePreview(panes, 40, 10, "layout", false, "1", false)
	if strings.TrimSpace(layout) == "" {
		t.Fatalf("renderPanePreview(layout) empty")
	}
	tiles := renderPanePreview(panes, 40, 10, "tiles", false, "", false)
	if strings.TrimSpace(tiles) == "" {
		t.Fatalf("renderPanePreview(tiles) empty")
	}
}

func TestRenderHelpers(t *testing.T) {
	if renderBadge(paneStatusRunning) == "" {
		t.Fatalf("renderBadge() empty")
	}
	if line := truncateLine("hello", 3); line == "" {
		t.Fatalf("truncateLine() empty")
	}
	if line := fitLine("hello", 3); line == "" {
		t.Fatalf("fitLine() empty")
	}
	if out := padLines("x", 4, 2); out == "" {
		t.Fatalf("padLines() empty")
	}
	if out := padRight("x", 3); out != "x  " {
		t.Fatalf("padRight() = %q", out)
	}
}

func TestPathHelpers(t *testing.T) {
	if pathOrDash("") != "-" {
		t.Fatalf("pathOrDash(empty)")
	}
	if out := tailLines([]string{"a", "b", "c"}, 2); len(out) != 2 {
		t.Fatalf("tailLines() = %#v", out)
	}
}

func TestViewStates(t *testing.T) {
	base := Model{
		Width:                    80,
		Height:                   24,
		HeaderLine:               "Peaky Panes",
		EmptyStateMessage:        "empty",
		DashboardColumns:         []DashboardColumn{{ProjectID: "proj", ProjectName: "Proj", ProjectPath: "", Panes: []DashboardPane{{ProjectID: "proj", ProjectName: "Proj", SessionName: "sess", Pane: Pane{Index: "0"}}}}},
		DashboardSelectedProject: "proj",
		Projects:                 []Project{{Name: "Proj"}},
		SidebarProject:           &Project{Name: "Proj"},
		SidebarSessions:          []Session{{Name: "sess", Status: sessionRunning}},
		PreviewSession:           &Session{Name: "sess", Status: sessionRunning, Panes: []Pane{{Index: "0"}}},
		QuickReplyInput:          textinput.New(),
		ProjectPicker:            list.New(nil, list.NewDefaultDelegate(), 10, 4),
		LayoutPicker:             list.New(nil, list.NewDefaultDelegate(), 10, 4),
		PaneSwapPicker:           list.New(nil, list.NewDefaultDelegate(), 10, 4),
		CommandPalette:           list.New(nil, list.NewDefaultDelegate(), 10, 4),
		ProjectRootInput:         textinput.New(),
		Rename:                   Rename{Input: textinput.New()},
	}

	cases := []int{
		viewHelp,
		viewConfirmKill,
		viewConfirmCloseProject,
		viewConfirmCloseAllProjects,
		viewConfirmRestart,
		viewRenameSession,
		viewRenamePane,
		viewProjectRootSetup,
		viewLayoutPicker,
		viewCommandPalette,
		viewDashboard,
	}
	for _, view := range cases {
		m := base
		m.ActiveView = view
		out := Render(m)
		if strings.TrimSpace(out) == "" {
			t.Fatalf("Render(view=%d) empty", view)
		}
	}
}

func TestSessionBadgeStatus(t *testing.T) {
	session := Session{Status: sessionRunning}
	if status := sessionBadgeStatus(session); status != paneStatusRunning {
		t.Fatalf("sessionBadgeStatus(running) = %v", status)
	}
	session = Session{Status: sessionStopped}
	if status := sessionBadgeStatus(session); status != paneStatusIdle {
		t.Fatalf("sessionBadgeStatus(stopped) = %v", status)
	}
	session = Session{ThumbnailLine: "done", ThumbnailStatus: paneStatusDone}
	if status := sessionBadgeStatus(session); status != paneStatusDone {
		t.Fatalf("sessionBadgeStatus(thumbnail) = %v", status)
	}
}
