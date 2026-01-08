package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestViewDashboardContentRenders(t *testing.T) {
	input := textinput.New()
	input.SetValue("hello")
	m := Model{
		Width:                    80,
		Height:                   24,
		HeaderLine:               "Peaky Panes",
		EmptyStateMessage:        "empty",
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
	if !strings.Contains(out, "hello") {
		t.Fatalf("viewDashboardContent() missing quick reply input")
	}
}

func TestRenderPaneLayout(t *testing.T) {
	panes := []Pane{
		{Index: "0", Title: "main", Left: 0, Top: 0, Width: 50, Height: 20, Active: true, Status: paneStatusRunning, ID: "p0"},
		{Index: "1", Title: "side", Left: 50, Top: 0, Width: 50, Height: 20, Active: false, Status: paneStatusIdle, ID: "p1"},
	}
	out := renderPaneLayout(panes, 40, 10, layoutPreviewContext{targetPane: "1"})
	if strings.TrimSpace(out) == "" {
		t.Fatalf("renderPaneLayout empty")
	}
}

func TestRenderPaneLayoutShowsResizeHandleGlyph(t *testing.T) {
	panes := []Pane{
		{Index: "0", Title: "left", Left: 0, Top: 0, Width: 500, Height: layout.LayoutBaseSize, Active: true, Status: paneStatusRunning, ID: "p0"},
		{Index: "1", Title: "right", Left: 500, Top: 0, Width: 500, Height: layout.LayoutBaseSize, Active: false, Status: paneStatusIdle, ID: "p1"},
	}
	preview := mouse.Rect{X: 0, Y: 0, W: 40, H: 10}
	rects := map[string]layout.Rect{
		"p0": {X: 0, Y: 0, W: 500, H: layout.LayoutBaseSize},
		"p1": {X: 500, Y: 0, W: 500, H: layout.LayoutBaseSize},
	}
	geom, ok := layoutgeom.Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	line, ok := layoutgeom.EdgeLineRect(geom, layoutgeom.EdgeRef{PaneID: "p0", Edge: sessiond.ResizeEdgeRight})
	if !ok || line.Empty() {
		t.Fatalf("expected divider line rect")
	}
	out := renderPaneLayout(panes, preview.W, preview.H, layoutPreviewContext{
		guides: []ResizeGuide{{X: line.X, Y: line.Y, W: line.W, H: line.H, Active: false}},
	})
	if !strings.Contains(out, "↔") {
		t.Fatalf("expected resize handle glyph in output")
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
		PekyDialogTitle:          "Peky",
		PekyDialogFooter:         "esc close",
		PekyDialogViewport:       viewport.New(10, 4),
	}
	base.PekyDialogViewport.SetContent("hello")

	cases := []int{
		viewHelp,
		viewConfirmKill,
		viewConfirmQuit,
		viewConfirmCloseProject,
		viewConfirmCloseAllProjects,
		viewConfirmRestart,
		viewRestartNotice,
		viewRenameSession,
		viewRenamePane,
		viewProjectRootSetup,
		viewLayoutPicker,
		viewCommandPalette,
		viewDashboard,
		viewPekyDialog,
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

func TestFooterRendersServerStatus(t *testing.T) {
	m := Model{
		Width:             80,
		Height:            24,
		ActiveView:        viewDashboard,
		HeaderLine:        "Peaky Panes",
		EmptyStateMessage: "empty",
		Projects:          []Project{{Name: "Proj"}},
		DashboardColumns: []DashboardColumn{{
			ProjectID:   "proj",
			ProjectName: "Proj",
			Panes: []DashboardPane{{
				ProjectID:   "proj",
				ProjectName: "Proj",
				SessionName: "sess",
				Pane:        Pane{Index: "0"},
			}},
		}},
		DashboardSelectedProject: "proj",
		SidebarProject:           &Project{Name: "Proj"},
		SidebarSessions:          []Session{{Name: "sess", Status: sessionRunning}},
		PreviewSession:           &Session{Name: "sess", Status: sessionRunning, Panes: []Pane{{Index: "0"}}},
		QuickReplyInput:          textinput.New(),
	}

	out := Render(m)
	lines := strings.Split(out, "\n")
	footer := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "quit") {
			footer = lines[i]
			break
		}
	}
	if footer == "" {
		t.Fatalf("expected footer line containing quit")
	}
	plain := strings.TrimRight(ansi.Strip(footer), " ")
	if !strings.HasSuffix(plain, "up") {
		t.Fatalf("footer=%q want suffix up", plain)
	}

	m.ServerStatus = "down"
	out = Render(m)
	lines = strings.Split(out, "\n")
	footer = ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "quit") {
			footer = lines[i]
			break
		}
	}
	if footer == "" {
		t.Fatalf("expected footer line containing quit")
	}
	plain = strings.TrimRight(ansi.Strip(footer), " ")
	if !strings.HasSuffix(plain, "down") {
		t.Fatalf("footer=%q want suffix down", plain)
	}

	m.ServerStatus = "restored"
	out = Render(m)
	lines = strings.Split(out, "\n")
	footer = ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "quit") {
			footer = lines[i]
			break
		}
	}
	if footer == "" {
		t.Fatalf("expected footer line containing quit")
	}
	plain = strings.TrimRight(ansi.Strip(footer), " ")
	if !strings.HasSuffix(plain, "restored") {
		t.Fatalf("footer=%q want suffix restored", plain)
	}
}

func TestRenderPadsToScreenSize(t *testing.T) {
	m := Model{
		Width:             60,
		Height:            16,
		ActiveView:        viewDashboard,
		HeaderLine:        "Peaky Panes",
		EmptyStateMessage: "empty",
		Projects:          []Project{{Name: "Proj"}},
		DashboardColumns: []DashboardColumn{{
			ProjectID:   "proj",
			ProjectName: "Proj",
			Panes: []DashboardPane{{
				ProjectID:   "proj",
				ProjectName: "Proj",
				SessionName: "sess",
				Pane:        Pane{Index: "0"},
			}},
		}},
		DashboardSelectedProject: "proj",
		SidebarProject:           &Project{Name: "Proj"},
		SidebarSessions:          []Session{{Name: "sess", Status: sessionRunning}},
		PreviewSession:           &Session{Name: "sess", Status: sessionRunning, Panes: []Pane{{Index: "0"}}},
		QuickReplyInput:          textinput.New(),
	}

	out := Render(m)
	lines := strings.Split(out, "\n")
	if len(lines) != m.Height {
		t.Fatalf("Render() lines=%d want=%d", len(lines), m.Height)
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w != m.Width {
			t.Fatalf("Render() line[%d] width=%d want=%d", i, w, m.Width)
		}
	}
}
