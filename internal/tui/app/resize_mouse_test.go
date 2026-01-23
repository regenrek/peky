package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
)

func TestResizeDragCommitUpdatesLayout(t *testing.T) {
	m := newLayoutResizeTestModel(t)
	session := m.selectedSession()
	if session == nil {
		t.Fatalf("expected selected session")
		return
	}
	engine := m.layoutEngines[session.Name]
	if engine == nil || engine.Tree == nil {
		t.Fatalf("expected layout engine")
	}
	before := engine.Tree.Rects()

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry")
	}
	edge := geom.Edges[0]
	startX := edge.HitRect.X + edge.HitRect.W/2
	startY := edge.HitRect.Y + edge.HitRect.H/2

	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: startX, Y: startY})
	if !m.resize.drag.active {
		t.Fatalf("expected resize drag active")
	}

	moveX := startX + 10
	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionMotion, X: moveX, Y: startY})
	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: moveX, Y: startY})

	if m.resize.drag.active {
		t.Fatalf("expected resize drag cleared")
	}
	after := m.layoutEngines[session.Name].Tree.Rects()
	if before[session.Panes[0].ID] == after[session.Panes[0].ID] {
		t.Fatalf("expected layout change after commit")
	}
}

func TestResizeDragEscCancelsLayoutChange(t *testing.T) {
	m := newLayoutResizeTestModel(t)
	session := m.selectedSession()
	if session == nil {
		t.Fatalf("expected selected session")
		return
	}
	engine := m.layoutEngines[session.Name]
	if engine == nil || engine.Tree == nil {
		t.Fatalf("expected layout engine")
	}
	before := engine.Tree.Rects()

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry")
	}
	edge := geom.Edges[0]
	startX := edge.HitRect.X + edge.HitRect.W/2
	startY := edge.HitRect.Y + edge.HitRect.H/2

	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: startX, Y: startY})
	if !m.resize.drag.active {
		t.Fatalf("expected resize drag active")
	}
	moveX := startX + 10
	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionMotion, X: moveX, Y: startY})

	_, _ = m.handleResizeKey(keyMsgFromTea(tea.KeyMsg{Type: tea.KeyEsc}))
	if m.resize.drag.active {
		t.Fatalf("expected resize drag canceled")
	}
	after := m.layoutEngines[session.Name].Tree.Rects()
	if before[session.Panes[0].ID] != after[session.Panes[0].ID] {
		t.Fatalf("expected layout unchanged after cancel")
	}
}

func TestResizeDragHorizontalEdgeUpdatesLayout(t *testing.T) {
	m := newLayoutResizeTestModelWithGrid(t, "2x1")
	session := m.selectedSession()
	if session == nil {
		t.Fatalf("expected selected session")
		return
	}
	engine := m.layoutEngines[session.Name]
	if engine == nil || engine.Tree == nil {
		t.Fatalf("expected layout engine")
	}
	before := engine.Tree.Rects()

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry")
	}
	var edge resizeEdgeRef
	found := false
	for _, candidate := range geom.Edges {
		if candidate.Ref.Edge == sessiond.ResizeEdgeDown {
			edge = candidate.Ref
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected horizontal resize edge")
	}
	hit, ok := layoutgeom.EdgeHitRect(geom, edge)
	if !ok {
		t.Fatalf("expected edge hit rect")
	}
	startX := hit.X + hit.W/2
	startY := hit.Y + hit.H/2

	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: startX, Y: startY})
	if !m.resize.drag.active {
		t.Fatalf("expected resize drag active")
	}
	moveY := startY + 8
	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionMotion, X: startX, Y: moveY})
	_, _ = m.handleResizeMouse(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: startX, Y: moveY})

	after := m.layoutEngines[session.Name].Tree.Rects()
	if before[session.Panes[0].ID] == after[session.Panes[0].ID] {
		t.Fatalf("expected layout change after horizontal drag")
	}
}

func newLayoutResizeTestModel(t *testing.T) *Model {
	return newLayoutResizeTestModelWithGrid(t, "1x2")
}

func newLayoutResizeTestModelWithGrid(t *testing.T, grid string) *Model {
	t.Helper()
	m := newTestModelLite()
	m.resize.snap = false
	session := m.selectedSession()
	if session == nil {
		t.Fatalf("expected selected session")
		return m
	}
	paneIDs := make([]string, 0, len(session.Panes))
	for _, pane := range session.Panes {
		paneIDs = append(paneIDs, pane.ID)
	}
	if len(paneIDs) < 2 {
		t.Fatalf("expected at least 2 panes")
	}
	tree, err := layout.BuildTree(&layout.LayoutConfig{Grid: grid}, paneIDs)
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}
	if m.layoutEngines == nil {
		m.layoutEngines = make(map[string]*layout.Engine)
	}
	m.layoutEngines[session.Name] = layout.NewEngine(tree)
	return m
}
