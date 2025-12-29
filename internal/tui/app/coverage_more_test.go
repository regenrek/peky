package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/agent"
	"github.com/regenrek/peakypanes/internal/userpath"
)

func TestConfigWatchState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".peakypanes.yml")
	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	state, ok := statConfigFile(path)
	if !ok || !state.exists || state.path != path {
		t.Fatalf("statConfigFile = %#v ok=%v", state, ok)
	}
	found := projectConfigStateForPath(dir)
	if !state.equal(found) {
		t.Fatalf("expected equal config state")
	}
	if state.equal(projectConfigState{}) {
		t.Fatalf("unexpected equality with empty state")
	}
}

func TestPaneStatusFromAgent(t *testing.T) {
	if paneStatusFromAgent(agent.StatusRunning) != PaneStatusRunning {
		t.Fatalf("paneStatusFromAgent running")
	}
	if paneStatusFromAgent(agent.StatusError) != PaneStatusError {
		t.Fatalf("paneStatusFromAgent error")
	}
	if paneStatusFromAgent(agent.StatusDone) != PaneStatusDone {
		t.Fatalf("paneStatusFromAgent done")
	}
	if paneStatusFromAgent(agent.StatusIdle) != PaneStatusIdle {
		t.Fatalf("paneStatusFromAgent idle")
	}
	if paneStatusFromAgent(agent.Status(99)) != PaneStatusIdle {
		t.Fatalf("paneStatusFromAgent default")
	}
}

func TestClassifyPaneFromLines(t *testing.T) {
	matcher, err := compileStatusMatcher(layout.StatusRegexConfig{
		Success: "done",
		Error:   "fail",
		Running: "run",
	})
	if err != nil {
		t.Fatalf("compileStatusMatcher: %v", err)
	}
	if status, ok := classifyPaneFromLines([]string{"run now"}, matcher); !ok || status != PaneStatusRunning {
		t.Fatalf("classifyPaneFromLines running: %v %v", status, ok)
	}
	if status, ok := classifyPaneFromLines([]string{"done"}, matcher); !ok || status != PaneStatusDone {
		t.Fatalf("classifyPaneFromLines done: %v %v", status, ok)
	}
	if status, ok := classifyPaneFromLines([]string{"fail"}, matcher); !ok || status != PaneStatusError {
		t.Fatalf("classifyPaneFromLines error: %v %v", status, ok)
	}
	if matchesRunning([]string{"run"}, matcher) != true {
		t.Fatalf("matchesRunning expected true")
	}
	if matchesRunning([]string{""}, matcher) != false {
		t.Fatalf("matchesRunning expected false")
	}
}

func TestPaneStatusForDeadIdleAndSummary(t *testing.T) {
	pane := PaneItem{Dead: true, DeadStatus: 1}
	if status, ok := paneStatusForDead(pane); !ok || status != PaneStatusError {
		t.Fatalf("paneStatusForDead error: %v %v", status, ok)
	}
	pane.DeadStatus = 0
	if status, ok := paneStatusForDead(pane); !ok || status != PaneStatusDone {
		t.Fatalf("paneStatusForDead done: %v %v", status, ok)
	}

	now := time.Now()
	if !paneIdle(PaneItem{LastActive: now.Add(-2 * time.Second)}, DashboardConfig{IdleThreshold: time.Second}, now) {
		t.Fatalf("paneIdle expected true")
	}

	line := paneSummaryLine(PaneItem{Preview: []string{"", "last"}}, 0)
	if line != "last" {
		t.Fatalf("paneSummaryLine preview = %q", line)
	}
	line = paneSummaryLine(PaneItem{Title: "title"}, 0)
	if line != "title" {
		t.Fatalf("paneSummaryLine title = %q", line)
	}
	line = paneSummaryLine(PaneItem{Command: "cmd"}, 0)
	if line != "cmd" {
		t.Fatalf("paneSummaryLine command = %q", line)
	}
	line = paneSummaryLine(PaneItem{Index: "3"}, 0)
	if line != "pane 3" {
		t.Fatalf("paneSummaryLine index = %q", line)
	}
}

func TestSelectionHelpers(t *testing.T) {
	filter := textinput.New()
	filter.SetValue("proj")
	m := &Model{
		filterInput: filter,
		data: DashboardData{
			Projects: []ProjectGroup{
				{ID: projectKey("", "proj"), Name: "proj"},
				{ID: projectKey("", "other"), Name: "other"},
			},
		},
	}
	sessions := []SessionItem{
		{Name: "proj session", Path: "/tmp/proj"},
		{Name: "misc", Path: "/tmp/misc"},
	}
	filtered := m.filteredSessions(sessions)
	if len(filtered) != 1 || filtered[0].Name != "proj session" {
		t.Fatalf("filteredSessions = %#v", filtered)
	}

	columns := []DashboardProjectColumn{
		{
			ProjectID:   projectKey("", "proj"),
			ProjectName: "proj",
			ProjectPath: "/tmp/proj",
			Panes: []DashboardPane{
				{ProjectID: projectKey("", "proj"), ProjectName: "proj", SessionName: "s1", Pane: PaneItem{Title: "build"}},
			},
		},
		{
			ProjectID:   projectKey("", "other"),
			ProjectName: "other",
			ProjectPath: "/tmp/other",
			Panes: []DashboardPane{
				{ProjectID: projectKey("", "other"), ProjectName: "other", SessionName: "s2", Pane: PaneItem{Title: "misc"}},
			},
		},
	}
	filteredCols := m.filteredDashboardColumns(columns)
	if len(filteredCols) != 2 || len(filteredCols[0].Panes) != 1 {
		t.Fatalf("filteredDashboardColumns = %#v", filteredCols)
	}

	if idx, ok := m.projectIndexForID(projectKey("", "proj")); !ok || idx != 0 {
		t.Fatalf("projectIndexFor = %d %v", idx, ok)
	}

	m.selection = selectionState{ProjectID: projectKey("", "other")}
	if idx := m.dashboardProjectIndex(columns); idx != 1 {
		t.Fatalf("dashboardProjectIndex by project = %d", idx)
	}
	m.selection = selectionState{ProjectID: "", Session: "s1"}
	if idx := m.dashboardProjectIndex(columns); idx != 0 {
		t.Fatalf("dashboardProjectIndex by session = %d", idx)
	}

	if got := sessionIndex(sessions, "misc"); got != 1 {
		t.Fatalf("sessionIndex = %d", got)
	}
	if wrapIndex(-1, 3) != 2 || wrapIndex(3, 3) != 0 {
		t.Fatalf("wrapIndex unexpected")
	}
}

func TestTerminalInputEncoding(t *testing.T) {
	if got := encodeKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}); string(got) != "a" {
		t.Fatalf("encodeKeyMsg runes = %q", string(got))
	}
	if got := encodeKeyMsg(tea.KeyMsg{Type: tea.KeySpace}); string(got) != " " {
		t.Fatalf("encodeKeyMsg space = %q", string(got))
	}
	if got := encodeKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("")}); len(got) != 0 {
		t.Fatalf("encodeKeyMsg empty runes should be empty")
	}
	if seq := ctrlSequence("ctrl+a"); len(seq) != 1 || seq[0] != 1 {
		t.Fatalf("ctrlSequence = %#v", seq)
	}
	if seq := ctrlSequence("ctrl+1"); seq != nil {
		t.Fatalf("ctrlSequence should reject digits")
	}
	if seq := altSequence("alt+x"); string(seq) != "\x1bx" {
		t.Fatalf("altSequence = %q", string(seq))
	}
}

func TestPaneSwapChoiceMethods(t *testing.T) {
	p := PaneSwapChoice{Label: "Pane 1", Desc: "demo"}
	if p.Title() != "Pane 1" || p.Description() != "demo" {
		t.Fatalf("pane swap choice fields")
	}
	if !strings.Contains(p.FilterValue(), "pane 1") || !strings.Contains(p.FilterValue(), "demo") {
		t.Fatalf("FilterValue = %q", p.FilterValue())
	}
}

func TestViewModelHelpers(t *testing.T) {
	pane := PaneItem{ID: "id", Index: "1", Title: "bash", Command: "bash", Active: true, Width: 10, Height: 5}
	session := SessionItem{Name: "sess", Status: StatusRunning, PaneCount: 1, ActivePane: "1", Panes: []PaneItem{pane}, Thumbnail: PaneSummary{Line: "sum", Status: PaneStatusRunning}}
	project := ProjectGroup{ID: projectKey("/tmp/proj", "proj"), Name: "proj", Path: "/tmp/proj", Sessions: []SessionItem{session}}

	view := toViewProject(project)
	if view.Name != "proj" || view.Path == "" {
		t.Fatalf("toViewProject = %#v", view)
	}
	if got := toViewSession(session); got.Name != "sess" || got.ActivePane != "1" {
		t.Fatalf("toViewSession = %#v", got)
	}
	if got := toViewPane(pane); got.ID != "id" || got.Index != "1" {
		t.Fatalf("toViewPane = %#v", got)
	}

	columns := []DashboardProjectColumn{{ProjectID: projectKey("/tmp/proj", "proj"), ProjectName: "proj", ProjectPath: "/tmp/proj", Panes: []DashboardPane{{ProjectID: projectKey("/tmp/proj", "proj"), ProjectName: "proj", ProjectPath: "/tmp/proj", SessionName: "sess", Pane: pane}}}}
	if got := toViewColumns(columns); len(got) != 1 || len(got[0].Panes) != 1 {
		t.Fatalf("toViewColumns = %#v", got)
	}

	if count := runningSessionsForProject([]ProjectGroup{project}, project.ID); count != 1 {
		t.Fatalf("runningSessionsForProject = %d", count)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "proj")
	if got := displayPath(path); got != userpath.ShortenUser(path) {
		t.Fatalf("displayPath = %q", got)
	}
}

func TestPaneLayoutHelpers(t *testing.T) {
	if dashboardPaneBlockHeight(2) != 6 {
		t.Fatalf("dashboardPaneBlockHeight unexpected")
	}
	panes := []PaneItem{{Left: 0, Top: 0, Width: 10, Height: 5}, {Left: 10, Top: 5, Width: 5, Height: 5}}
	maxW, maxH := paneBounds(panes)
	if maxW != 15 || maxH != 10 {
		t.Fatalf("paneBounds = %d %d", maxW, maxH)
	}
	x, y, w, h := scalePane(PaneItem{Left: 0, Top: 0, Width: 5, Height: 5}, 10, 10, 20, 10)
	if w < 2 || h < 2 || x < 0 || y < 0 {
		t.Fatalf("scalePane = %d %d %d %d", x, y, w, h)
	}
}

func TestSessionExpandedAndMouseFilter(t *testing.T) {
	var m *Model
	if !m.sessionExpanded("any") {
		t.Fatalf("nil model should default expanded")
	}
	model := &Model{expandedSessions: map[string]bool{"sess": false}}
	if model.sessionExpanded("sess") {
		t.Fatalf("expected sessionExpanded false")
	}

	filter := NewMouseMotionFilter()
	msg := tea.MouseMsg{Action: tea.MouseActionMotion, X: 1, Y: 2}
	filter.lastX = 9
	filter.lastY = 9
	if got := filter.Filter(nil, msg); got != nil {
		t.Fatalf("expected nil for nil model")
	}
	if filter.lastX != -1 || filter.lastY != -1 {
		t.Fatalf("expected filter reset")
	}

	model = modelWithMouseMotion()
	msg = tea.MouseMsg{Action: tea.MouseActionMotion, X: 3, Y: 4}
	if got := filter.Filter(model, msg); got == nil {
		t.Fatalf("expected mouse msg returned")
	}
	if got := filter.Filter(model, msg); got != nil {
		t.Fatalf("expected duplicate motion filtered")
	}
	filter.lastAt = time.Now()
	if got := filter.Filter(model, tea.MouseMsg{Action: tea.MouseActionMotion, X: 4, Y: 5}); got != nil {
		t.Fatalf("expected throttle to drop motion")
	}
}

func modelWithMouseMotion() *Model {
	m := &Model{
		client:          &sessiond.Client{},
		state:           StateDashboard,
		tab:             TabProject,
		terminalFocus:   true,
		paneMouseMotion: map[string]bool{"pane1": true},
		data: DashboardData{
			Projects: []ProjectGroup{
				{
					Name: "proj",
					Sessions: []SessionItem{
						{Name: "sess", Panes: []PaneItem{{ID: "pane1", Index: "1"}}},
					},
				},
			},
		},
	}
	m.selection = selectionState{ProjectID: projectKey("", "proj"), Session: "sess", Pane: "1"}
	return m
}

func TestEncodeKeyMsgSequences(t *testing.T) {
	seq := encodeKeyMsg(tea.KeyMsg{Type: tea.KeyTab})
	if string(seq) != "\t" {
		t.Fatalf("encodeKeyMsg tab = %q", string(seq))
	}
}

func TestToViewProjects(t *testing.T) {
	projects := []ProjectGroup{{ID: projectKey("/tmp/proj", "proj"), Name: "proj", Path: "/tmp/proj"}}
	got := toViewProjects(projects)
	if len(got) != 1 || got[0].Name != "proj" {
		t.Fatalf("toViewProjects = %#v", got)
	}
	if toViewProjectPtr(nil) != nil {
		t.Fatalf("expected nil project pointer")
	}
}
