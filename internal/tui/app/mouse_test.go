package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/theme"
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
	if m.hardRaw {
		t.Fatalf("hardRaw should remain false after single click")
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

func TestProjectPickerMouseClickSelectsItem(t *testing.T) {
	m := newTestModelLite()
	m.state = StateProjectPicker
	m.applyWindowSize(tea.WindowSizeMsg{Width: 120, Height: 40})

	root := t.TempDir()
	dir1 := filepath.Join(root, "one")
	dir2 := filepath.Join(root, "two")
	dir3 := filepath.Join(root, "three")
	for _, dir := range []string{dir1, dir2, dir3} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	items := []list.Item{
		picker.ProjectItem{Name: "one", Path: dir1},
		picker.ProjectItem{Name: "two", Path: dir2},
		picker.ProjectItem{Name: "three", Path: dir3},
	}
	_ = m.projectPicker.SetItems(items)
	m.projectPicker.Select(0)

	itemHeight, rowHeight := picker.ProjectPickerRowMetrics()
	padLeft := theme.App.GetPaddingLeft()
	padTop := theme.App.GetPaddingTop()
	titleHeight := sectionHeight(1, m.projectPicker.Styles.TitleBar)
	statusHeight := sectionHeight(1, m.projectPicker.Styles.StatusBar)

	x := padLeft + 3
	y := padTop + titleHeight + statusHeight + rowHeight*1
	// Clicking on the spacing line should do nothing.
	_, _ = m.updateProjectPickerMouse(tea.MouseMsg{X: x, Y: y + itemHeight, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if m.state != StateProjectPicker {
		t.Fatalf("state=%v want %v after spacing click", m.state, StateProjectPicker)
	}
	if m.projectPicker.Index() != 0 {
		t.Fatalf("index=%d want 0 after spacing click", m.projectPicker.Index())
	}

	_, _ = m.updateProjectPickerMouse(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})

	if m.state != StateDashboard {
		t.Fatalf("state=%v want %v after click open", m.state, StateDashboard)
	}
	if m.selection.ProjectID != projectKey(dir2, "two") {
		t.Fatalf("projectID=%q want %q", m.selection.ProjectID, projectKey(dir2, "two"))
	}
}

func TestProjectPickerMouseWheelMovesSelection(t *testing.T) {
	m := newTestModelLite()
	m.state = StateProjectPicker
	m.applyWindowSize(tea.WindowSizeMsg{Width: 120, Height: 40})

	_ = m.projectPicker.SetItems([]list.Item{
		picker.ProjectItem{Name: "one", Path: "/tmp/one"},
		picker.ProjectItem{Name: "two", Path: "/tmp/two"},
	})
	m.projectPicker.Select(0)

	_, _ = m.updateProjectPickerMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	if m.projectPicker.Index() != 1 {
		t.Fatalf("index=%d want 1", m.projectPicker.Index())
	}
	_, _ = m.updateProjectPickerMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	if m.projectPicker.Index() != 0 {
		t.Fatalf("index=%d want 0", m.projectPicker.Index())
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

func TestMouseDragStarts(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}

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
		t.Fatalf("expected terminalMouseDrag true when press starts")
	}
}

func TestMousePaneTopbarClickOpensPaneDetailsDialog(t *testing.T) {
	m := newTestModelLite()
	m.settings.PaneTopbar.Enabled = true
	m.data.Projects[0].Sessions[0].Panes = []PaneItem{{
		ID:     "p1",
		Index:  "1",
		Title:  "one",
		Cwd:    "/tmp/demo",
		Left:   0,
		Top:    0,
		Width:  layout.LayoutBaseSize,
		Height: layout.LayoutBaseSize,
	}}
	m.selection = selectionState{ProjectID: m.data.Projects[0].ID, Session: m.data.Projects[0].Sessions[0].Name, Pane: "1"}

	hits := m.paneHits()
	if len(hits) == 0 {
		t.Fatalf("expected pane hits")
	}
	hit := hits[0]
	if hit.Topbar.Empty() {
		t.Fatalf("expected topbar rect")
	}
	msg := tea.MouseMsg{X: hit.Topbar.X + 1, Y: hit.Topbar.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.state != StatePekyDialog {
		t.Fatalf("state=%v want %v", m.state, StatePekyDialog)
	}
	if m.pekyDialogTitle != "Pane details" {
		t.Fatalf("dialog title=%q", m.pekyDialogTitle)
	}
	if !strings.Contains(m.pekyViewport.View(), "CWD:") {
		t.Fatalf("expected CWD in dialog: %q", m.pekyViewport.View())
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

func TestMouseQuickReplyClickExitsHardRaw(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.setHardRaw(true)

	rect, ok := m.quickReplyRect()
	if !ok {
		t.Fatalf("quick reply rect unavailable")
	}
	msg := tea.MouseMsg{X: rect.X + 2, Y: rect.Y + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.hardRaw {
		t.Fatalf("hardRaw should be false after quick reply click")
	}
	m.quickReplyInput.SetValue("")
	m.updateDashboard(keyRune('x'))
	if got := m.quickReplyInput.Value(); got != "x" {
		t.Fatalf("quickReplyInput=%q want %q", got, "x")
	}
}

func TestMouseQuickReplySelectionCopiesToClipboard(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.quickReplyInput.SetValue("hello world")

	var copied string
	prev := writeClipboard
	writeClipboard = func(text string) error {
		copied = text
		return nil
	}
	defer func() { writeClipboard = prev }()

	inputRect, ok := m.quickReplyInputBounds()
	if !ok || inputRect.Empty() {
		t.Fatalf("quick reply input rect unavailable")
	}
	press := tea.MouseMsg{X: inputRect.X, Y: inputRect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	motion := tea.MouseMsg{X: inputRect.X + 5, Y: inputRect.Y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}
	release := tea.MouseMsg{X: inputRect.X + 5, Y: inputRect.Y, Action: tea.MouseActionRelease, Button: tea.MouseButtonNone}

	_, _ = m.updateDashboardMouse(press)
	_, _ = m.updateDashboardMouse(motion)
	_, _ = m.updateDashboardMouse(release)

	if copied != "hello" {
		t.Fatalf("clipboard=%q want %q", copied, "hello")
	}
	if m.quickReplyMouseSel.active {
		t.Fatalf("expected quick reply selection cleared")
	}
}

func TestMouseQuickReplySelectionDoesNotCopyEmptyRange(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.quickReplyInput.SetValue("hello")

	called := false
	prev := writeClipboard
	writeClipboard = func(text string) error {
		called = true
		return nil
	}
	defer func() { writeClipboard = prev }()

	inputRect, ok := m.quickReplyInputBounds()
	if !ok || inputRect.Empty() {
		t.Fatalf("quick reply input rect unavailable")
	}
	press := tea.MouseMsg{X: inputRect.X, Y: inputRect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	release := tea.MouseMsg{X: inputRect.X, Y: inputRect.Y, Action: tea.MouseActionRelease, Button: tea.MouseButtonNone}

	_, _ = m.updateDashboardMouse(press)
	_, _ = m.updateDashboardMouse(release)

	if called {
		t.Fatalf("expected clipboard not called for empty selection")
	}
	if m.quickReplyMouseSel.active {
		t.Fatalf("expected quick reply selection cleared")
	}
}

func TestMouseQuickReplySelectionDoesNotCopyWhitespace(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.quickReplyInput.SetValue("  hi")

	var copied string
	prev := writeClipboard
	writeClipboard = func(text string) error {
		copied = text
		return nil
	}
	defer func() { writeClipboard = prev }()

	inputRect, ok := m.quickReplyInputBounds()
	if !ok || inputRect.Empty() {
		t.Fatalf("quick reply input rect unavailable")
	}
	press := tea.MouseMsg{X: inputRect.X, Y: inputRect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	motion := tea.MouseMsg{X: inputRect.X + 2, Y: inputRect.Y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}
	release := tea.MouseMsg{X: inputRect.X + 2, Y: inputRect.Y, Action: tea.MouseActionRelease, Button: tea.MouseButtonNone}

	_, _ = m.updateDashboardMouse(press)
	_, _ = m.updateDashboardMouse(motion)
	_, _ = m.updateDashboardMouse(release)

	if copied != "" {
		t.Fatalf("expected no clipboard write for whitespace selection, got %q", copied)
	}
	if m.toast.Text != "" {
		t.Fatalf("expected no toast for whitespace selection, got %q", m.toast.Text)
	}
}

func TestMouseQuickReplySelectionClipboardErrorShowsToast(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.quickReplyInput.SetValue("hello world")

	prev := writeClipboard
	writeClipboard = func(text string) error {
		return errors.New("boom")
	}
	defer func() { writeClipboard = prev }()

	inputRect, ok := m.quickReplyInputBounds()
	if !ok || inputRect.Empty() {
		t.Fatalf("quick reply input rect unavailable")
	}
	press := tea.MouseMsg{X: inputRect.X, Y: inputRect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	motion := tea.MouseMsg{X: inputRect.X + 5, Y: inputRect.Y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone}
	release := tea.MouseMsg{X: inputRect.X + 5, Y: inputRect.Y, Action: tea.MouseActionRelease, Button: tea.MouseButtonNone}

	_, _ = m.updateDashboardMouse(press)
	_, _ = m.updateDashboardMouse(motion)
	_, _ = m.updateDashboardMouse(release)

	if m.toast.Text != "Copy failed" {
		t.Fatalf("toast=%q want %q", m.toast.Text, "Copy failed")
	}
	if m.toast.Level != toastWarning {
		t.Fatalf("toastLevel=%v want %v", m.toast.Level, toastWarning)
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

	session := findSessionByName(m.data.Projects, "alpha-1")
	if session == nil {
		t.Fatalf("expected session alpha-1")
		return
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

	session := findSessionByName(m.data.Projects, "alpha-1")
	if session == nil {
		t.Fatalf("expected session alpha-1")
		return
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
