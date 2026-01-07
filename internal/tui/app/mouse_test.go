package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestMouseSingleClickSelectsPane(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)

	hits := m.dashboardPaneHits()
	if len(hits) == 0 {
		t.Fatalf("expected dashboard pane hits")
	}
	hit := hits[0]
	msg := tea.MouseMsg{X: hit.Outer.X + 1, Y: hit.Outer.Y + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.selection != selectionFromMouse(hit.Selection) {
		t.Fatalf("selection=%#v want %#v", m.selection, hit.Selection)
	}
	if m.terminalFocus {
		t.Fatalf("terminalFocus should remain false after single click")
	}
}

func TestMouseDoubleClickEntersFocus(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)

	hits := m.dashboardPaneHits()
	if len(hits) == 0 {
		t.Fatalf("expected dashboard pane hits")
	}
	hit := hits[0]
	msg := tea.MouseMsg{X: hit.Outer.X + 1, Y: hit.Outer.Y + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)
	if m.terminalFocus {
		t.Fatalf("terminalFocus should remain false after first click")
	}

	_, _ = m.updateDashboardMouse(msg)
	if !m.terminalFocus {
		t.Fatalf("terminalFocus should be true after double click")
	}
}

func TestEscKeepsTerminalFocus(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.setTerminalFocus(true)

	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})
	if !m.terminalFocus {
		t.Fatalf("terminalFocus should remain true after esc")
	}
}

func TestMouseHeaderClickSwitchesToProjectTab(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)

	rect, ok := findHeaderRect(t, m, headerPartProject, projectKey("/tmp/beta", "Beta"))
	if !ok {
		t.Fatalf("project header rect not found")
	}
	msg := tea.MouseMsg{X: rect.X + 1, Y: rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.tab != TabProject {
		t.Fatalf("tab=%v want %v", m.tab, TabProject)
	}
	if m.selection.ProjectID != projectKey("/tmp/beta", "Beta") {
		t.Fatalf("selection project=%q want %q", m.selection.ProjectID, projectKey("/tmp/beta", "Beta"))
	}
}

func TestMouseHeaderClickSwitchesToDashboardTab(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)
	m.tab = TabProject
	m.selection.ProjectID = projectKey("/tmp/beta", "Beta")

	rect, ok := findHeaderRect(t, m, headerPartDashboard, "")
	if !ok {
		t.Fatalf("dashboard header rect not found")
	}
	msg := tea.MouseMsg{X: rect.X + 1, Y: rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.tab != TabDashboard {
		t.Fatalf("tab=%v want %v", m.tab, TabDashboard)
	}
}

func TestMouseHeaderClickNewOpensProjectPicker(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)
	m.settings.ProjectRoots = []string{t.TempDir()}

	rect, ok := findHeaderRect(t, m, headerPartNew, "")
	if !ok {
		t.Fatalf("new header rect not found")
	}
	msg := tea.MouseMsg{X: rect.X + 1, Y: rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.state != StateProjectPicker {
		t.Fatalf("state=%v want %v", m.state, StateProjectPicker)
	}
}

func TestMouseReleaseClearsTerminalDrag(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.terminalMouseDrag = true

	m.updateTerminalMouseDrag(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonNone})

	if m.terminalMouseDrag {
		t.Fatalf("terminalMouseDrag should be false after release")
	}
}

func TestMouseDragStartsWithoutTerminalFocus(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}
	m.terminalFocus = false

	hits := m.paneHits()
	if len(hits) == 0 {
		t.Fatalf("expected pane hits")
	}
	hit := hits[0]
	if hit.Content.Empty() {
		t.Fatalf("expected pane content rect")
	}

	msg := tea.MouseMsg{
		X:      hit.Content.X + hit.Content.W/2,
		Y:      hit.Content.Y + hit.Content.H/2,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	m.updateTerminalMouseDrag(msg)

	if !m.terminalMouseDrag {
		t.Fatalf("expected terminalMouseDrag true when press starts without focus")
	}
}

func seedMouseTestData(m *Model) {
	m.tab = TabDashboard
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectKey(m.configPath, "Proj"),
		Name: "Proj",
		Path: m.configPath,
		Sessions: []SessionItem{{
			Name:       "sess",
			Status:     StatusRunning,
			ActivePane: "1",
			Panes: []PaneItem{{
				ID:    "pane-1",
				Index: "1",
				Title: "pane",
			}},
		}},
	}}}
	m.selection = selectionState{}
}

func seedHeaderTestData(m *Model) {
	m.tab = TabDashboard
	m.data = DashboardData{Projects: []ProjectGroup{
		{
			ID:   projectKey("/tmp/alpha", "Alpha"),
			Name: "Alpha",
			Path: "/tmp/alpha",
			Sessions: []SessionItem{{
				Name:   "alpha",
				Status: StatusRunning,
				Panes: []PaneItem{{
					ID:    "pane-a",
					Index: "1",
					Title: "pane",
				}},
			}},
		},
		{
			ID:   projectKey("/tmp/beta", "Beta"),
			Name: "Beta",
			Path: "/tmp/beta",
			Sessions: []SessionItem{{
				Name:   "beta",
				Status: StatusRunning,
				Panes: []PaneItem{{
					ID:    "pane-b",
					Index: "1",
					Title: "pane",
				}},
			}},
		},
	}}
	m.selection = selectionState{ProjectID: projectKey("/tmp/alpha", "Alpha"), Session: "alpha", Pane: "1"}
}

func findHeaderRect(t *testing.T, m *Model, kind headerPartKind, projectID string) (mouse.Rect, bool) {
	t.Helper()
	wantKind, ok := headerHitKind(kind)
	if !ok {
		t.Fatalf("headerHitKind(%v) not mapped", kind)
	}
	for _, hit := range m.headerHitRects() {
		if hit.Hit.Kind != wantKind {
			continue
		}
		if wantKind == mouse.HeaderProject && hit.Hit.ProjectID != projectID {
			continue
		}
		return hit.Rect, true
	}
	return mouse.Rect{}, false
}

func TestMouseQuickReplyClickExitsTerminalFocus(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.setTerminalFocus(true)

	rect, ok := m.quickReplyRect()
	if !ok {
		t.Fatalf("quick reply rect unavailable")
	}
	msg := tea.MouseMsg{X: rect.X + 2, Y: rect.Y + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.terminalFocus {
		t.Fatalf("terminalFocus should be false after quick reply click")
	}
	m.quickReplyInput.SetValue("")
	m.updateDashboard(keyRune('x'))
	if got := m.quickReplyInput.Value(); got != "x" {
		t.Fatalf("quickReplyInput=%q want %q", got, "x")
	}
}

func TestMouseMotionSetsCursorShapeTextOverDashboardBody(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	var emitted string
	m.oscEmit = func(seq string) { emitted = seq }

	body, ok := m.dashboardBodyRect()
	if !ok {
		t.Fatalf("dashboard body rect unavailable")
	}
	msg := tea.MouseMsg{X: body.X + 1, Y: body.Y + 1, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}

	_, _ = m.updateDashboardMouse(msg)

	if emitted != "\x1b]22;text\x07" {
		t.Fatalf("osc=%q want %q", emitted, "\x1b]22;text\x07")
	}
}

func TestMouseMotionSetsCursorShapePointerOverHeader(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	var emitted string
	m.oscEmit = func(seq string) { emitted = seq }

	header, ok := m.headerRect()
	if !ok {
		t.Fatalf("header rect unavailable")
	}
	msg := tea.MouseMsg{X: header.X + 1, Y: header.Y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}

	_, _ = m.updateDashboardMouse(msg)

	if emitted != "\x1b]22;pointer\x07" {
		t.Fatalf("osc=%q want %q", emitted, "\x1b]22;pointer\x07")
	}
}

func TestMouseMotionSetsCursorShapeTextOverQuickReply(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	var emitted string
	m.oscEmit = func(seq string) { emitted = seq }

	rect, ok := m.quickReplyRect()
	if !ok || rect.Empty() {
		t.Fatalf("quick reply rect unavailable")
	}
	msg := tea.MouseMsg{X: rect.X + 2, Y: rect.Y + 1, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}

	_, _ = m.updateDashboardMouse(msg)

	if emitted != "\x1b]22;text\x07" {
		t.Fatalf("osc=%q want %q", emitted, "\x1b]22;text\x07")
	}
}

func TestMouseMotionSetsCursorShapeColResizeOverDivider(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}
	m.tab = TabProject
	m.state = StateDashboard
	m.terminalFocus = false

	session := findSessionByName(m.data.Projects, "alpha-1")
	if session == nil {
		t.Fatalf("expected session alpha-1")
	}
	session.LayoutTree = &layout.TreeSnapshot{
		Root: layout.NodeSnapshot{
			Axis: layout.AxisHorizontal,
			Size: layout.LayoutBaseSize,
			Children: []layout.NodeSnapshot{
				{PaneID: "p1", Size: 500},
				{PaneID: "p2", Size: 500},
			},
		},
	}
	m.syncLayoutEngines()

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry")
	}
	var targetRect mouse.Rect
	for _, edge := range geom.Edges {
		if edge.Axis != layoutgeom.SegmentVertical {
			continue
		}
		targetRect = edge.HitRect
		break
	}
	if targetRect.Empty() {
		t.Fatalf("expected vertical edge hit rect")
	}

	var emitted string
	m.oscEmit = func(seq string) { emitted = seq }
	msg := tea.MouseMsg{
		X:      targetRect.X + targetRect.W/2,
		Y:      targetRect.Y + targetRect.H/2,
		Action: tea.MouseActionMotion,
		Button: tea.MouseButtonNone,
	}
	_, _ = m.updateDashboardMouse(msg)

	if emitted != "\x1b]22;col-resize\x07\x1b]22;ew-resize\x07" {
		t.Fatalf("osc=%q want %q", emitted, "\x1b]22;col-resize\x07\x1b]22;ew-resize\x07")
	}
}

func TestMouseMotionSetsCursorShapeRowResizeOverDivider(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}
	m.tab = TabProject
	m.state = StateDashboard
	m.terminalFocus = false

	session := findSessionByName(m.data.Projects, "alpha-1")
	if session == nil {
		t.Fatalf("expected session alpha-1")
	}
	session.LayoutTree = &layout.TreeSnapshot{
		Root: layout.NodeSnapshot{
			Axis: layout.AxisVertical,
			Size: layout.LayoutBaseSize,
			Children: []layout.NodeSnapshot{
				{PaneID: "p1", Size: 500},
				{PaneID: "p2", Size: 500},
			},
		},
	}
	m.syncLayoutEngines()

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry")
	}
	var targetRect mouse.Rect
	for _, edge := range geom.Edges {
		if edge.Axis != layoutgeom.SegmentHorizontal {
			continue
		}
		targetRect = edge.HitRect
		break
	}
	if targetRect.Empty() {
		t.Fatalf("expected horizontal edge hit rect")
	}

	var emitted string
	m.oscEmit = func(seq string) { emitted = seq }
	msg := tea.MouseMsg{
		X:      targetRect.X + targetRect.W/2,
		Y:      targetRect.Y + targetRect.H/2,
		Action: tea.MouseActionMotion,
		Button: tea.MouseButtonNone,
	}
	_, _ = m.updateDashboardMouse(msg)

	if emitted != "\x1b]22;row-resize\x07\x1b]22;ns-resize\x07" {
		t.Fatalf("osc=%q want %q", emitted, "\x1b]22;row-resize\x07\x1b]22;ns-resize\x07")
	}
}
