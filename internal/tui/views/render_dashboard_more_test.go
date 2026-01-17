package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/regenrek/peakypanes/internal/tui/icons"
)

type testItem string

func (t testItem) FilterValue() string { return string(t) }

func TestSidebarEmptyStates(t *testing.T) {
	m := Model{Width: 80, Height: 24}
	out := m.viewSidebar(30, 6)
	if !strings.Contains(out, "No projects") {
		t.Fatalf("expected no projects message, got %q", out)
	}

	m.SidebarProject = &Project{Name: "demo"}
	out = m.viewSidebar(30, 6)
	if !strings.Contains(out, "No sessions") {
		t.Fatalf("expected no sessions message, got %q", out)
	}
}

func TestSidebarSessionsAndFilter(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("filter")

	m := Model{
		Width:            80,
		Height:           24,
		SidebarProject:   &Project{Name: "demo"},
		SidebarSessions:  []Session{{Name: "sess", PaneCount: 2, ActivePane: "1", Panes: []Pane{{Index: "1", Title: "alpha"}, {Index: "2", Title: "beta"}}}},
		SelectionSession: "sess",
		SelectionPane:    "2",
		ExpandedSessions: map[string]bool{"sess": true},
		FilterInput:      ti,
	}
	out := m.viewSidebar(40, 8)
	if !strings.Contains(out, "sess") {
		t.Fatalf("expected session name in sidebar, got %q", out)
	}
	if !strings.Contains(out, "Filter:") {
		t.Fatalf("expected filter line, got %q", out)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("expected pane index in sidebar, got %q", out)
	}
}

func TestViewProjectBody(t *testing.T) {
	m := Model{
		SidebarProject:  &Project{Name: "demo"},
		SidebarSessions: []Session{{Name: "sess", PaneCount: 1}},
		PreviewSession:  &Session{Name: "sess"},
	}
	out := m.viewProjectBody(40, 8)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected project body output")
	}
}

func TestSidebarHelpers(t *testing.T) {
	m := Model{SelectionSession: "s", SelectionPane: "p"}
	s := Session{Name: "s", ActivePane: "a", Panes: []Pane{{Index: "1"}}}
	if got := m.selectedPaneForSession(s); got != "p" {
		t.Fatalf("selectedPaneForSession = %q", got)
	}
	if appendSidebarGap("x", false) != "x\n" {
		t.Fatalf("appendSidebarGap did not add newline")
	}
	if appendSidebarGap("x", true) != "x" {
		t.Fatalf("appendSidebarGap should keep last line")
	}
	m.FilterActive = true
	if !m.shouldRenderFilterLine() {
		t.Fatalf("expected filter line when filter active")
	}
	m.FilterActive = false
	m.FilterInput = textinput.New()
	if m.shouldRenderFilterLine() {
		t.Fatalf("did not expect filter line without input")
	}
	if !m.sessionExpanded("missing") {
		t.Fatalf("expected sessionExpanded default true")
	}
	m.ExpandedSessions = map[string]bool{"sess": false}
	if m.sessionExpanded("sess") {
		t.Fatalf("expected sessionExpanded false for sess")
	}
}

func TestRenderDashboardPaneTileAndPreviewLines(t *testing.T) {
	pane := DashboardPane{
		SessionName: "sess",
		Pane: Pane{
			Index:       "1",
			Title:       "vim",
			Command:     "vim",
			Preview:     []string{"one", "two", "three"},
			SummaryLine: "summary",
		},
	}
	lines := panePreviewLinesWithWidth(pane, 2, 10, true)
	if len(lines) != 2 || !strings.Contains(lines[0], "two") {
		t.Fatalf("unexpected preview lines: %#v", lines)
	}
	pane.Pane.Preview = nil
	lines = panePreviewLinesWithWidth(pane, 1, 10, true)
	if len(lines) != 1 || !strings.Contains(lines[0], "summary") {
		t.Fatalf("expected summary preview, got %#v", lines)
	}

	out := renderDashboardPaneTile(pane, 24, 6, 2, true, paneIconContext{set: icons.Active(), size: icons.ActiveSize()})
	if !strings.Contains(out, "sess") && !strings.Contains(out, "vim") {
		t.Fatalf("expected pane tile content, got %q", out)
	}
}

func TestViewConfirmClosePane(t *testing.T) {
	m := Model{
		Width:  80,
		Height: 24,
		ConfirmClosePane: ConfirmClosePane{
			Title:   "shell",
			Session: "sess",
			Running: true,
		},
	}
	out := m.viewConfirmClosePane()
	if !strings.Contains(out, "Close Pane") || !strings.Contains(out, "shell") || !strings.Contains(out, "sess") {
		t.Fatalf("unexpected confirm close pane output: %q", out)
	}
}

func TestViewPanePickers(t *testing.T) {
	m := Model{Width: 80, Height: 24}
	out := m.viewPaneSplitPicker()
	if !strings.Contains(out, "Add Pane") {
		t.Fatalf("expected split picker output, got %q", out)
	}

	items := []list.Item{testItem("one")}
	picker := list.New(items, list.NewDefaultDelegate(), 20, 5)
	m.PaneSwapPicker = picker
	out = m.viewPaneSwapPicker()
	if out == "" {
		t.Fatalf("expected swap picker output")
	}
}
