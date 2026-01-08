//go:build integration

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestResizeDragCommitIntegration(t *testing.T) {
	m := newTestModel(t)
	snap := startSessionWithGrid(t, m, "resize-drag", "1x2")
	refreshModel(t, m)
	selectSessionForTest(t, m, snap.Name)
	m.settings.Resize.MouseApply = ResizeMouseApplyLive
	m.resize.snap = false

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry edges")
	}
	edge := geom.Edges[0]
	startX := edge.HitRect.X + edge.HitRect.W/2
	startY := edge.HitRect.Y + edge.HitRect.H/2

	cmd, handled := m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      startX,
		Y:      startY,
	})
	runHandledCmd(t, cmd, handled)

	moveX, moveY := startX, startY
	switch edge.Ref.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		moveX += 6
	default:
		moveY += 3
	}
	cmd, handled = m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionMotion,
		X:      moveX,
		Y:      moveY,
	})
	runHandledCmd(t, cmd, handled)
	cmd, handled = m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      moveX,
		Y:      moveY,
	})
	runHandledCmd(t, cmd, handled)

	after := sessionSnapshot(t, m, snap.Name)
	beforePane := findSnapshotPane(t, snap, edge.Ref.PaneID)
	afterPane := findSnapshotPane(t, after, edge.Ref.PaneID)
	switch edge.Ref.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		if beforePane.Width == afterPane.Width {
			t.Fatalf("expected width change after drag")
		}
	default:
		if beforePane.Height == afterPane.Height {
			t.Fatalf("expected height change after drag")
		}
	}
}

func TestResizeDragCommitModeIntegration(t *testing.T) {
	m := newTestModel(t)
	snap := startSessionWithGrid(t, m, "resize-commit-mode", "1x2")
	refreshModel(t, m)
	selectSessionForTest(t, m, snap.Name)
	m.settings.Resize.MouseApply = ResizeMouseApplyCommit
	m.resize.snap = false

	before := sessionSnapshot(t, m, snap.Name)

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry edges")
	}
	edge := geom.Edges[0]
	startX := edge.HitRect.X + edge.HitRect.W/2
	startY := edge.HitRect.Y + edge.HitRect.H/2

	cmd, handled := m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      startX,
		Y:      startY,
	})
	runHandledCmd(t, cmd, handled)

	moveX, moveY := startX, startY
	switch edge.Ref.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		moveX += 6
	default:
		moveY += 3
	}
	cmd, handled = m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionMotion,
		X:      moveX,
		Y:      moveY,
	})
	runHandledCmd(t, cmd, handled)

	mid := sessionSnapshot(t, m, snap.Name)
	beforePane := findSnapshotPane(t, before, edge.Ref.PaneID)
	midPane := findSnapshotPane(t, mid, edge.Ref.PaneID)
	switch edge.Ref.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		if beforePane.Width != midPane.Width {
			t.Fatalf("expected width unchanged before release")
		}
	default:
		if beforePane.Height != midPane.Height {
			t.Fatalf("expected height unchanged before release")
		}
	}

	cmd, handled = m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      moveX,
		Y:      moveY,
	})
	runHandledCmd(t, cmd, handled)

	after := sessionSnapshot(t, m, snap.Name)
	afterPane := findSnapshotPane(t, after, edge.Ref.PaneID)
	switch edge.Ref.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		if beforePane.Width == afterPane.Width {
			t.Fatalf("expected width change after release")
		}
	default:
		if beforePane.Height == afterPane.Height {
			t.Fatalf("expected height change after release")
		}
	}
}

func TestResizeDragHorizontalIntegration(t *testing.T) {
	m := newTestModel(t)
	snap := startSessionWithGrid(t, m, "resize-drag-horizontal", "2x1")
	refreshModel(t, m)
	selectSessionForTest(t, m, snap.Name)
	m.settings.Resize.MouseApply = ResizeMouseApplyLive
	m.resize.snap = false

	geom, ok := m.resizeGeometry()
	if !ok || len(geom.Edges) == 0 {
		t.Fatalf("expected resize geometry edges")
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

	cmd, handled := m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      startX,
		Y:      startY,
	})
	runHandledCmd(t, cmd, handled)

	moveY := startY + 5
	cmd, handled = m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionMotion,
		X:      startX,
		Y:      moveY,
	})
	runHandledCmd(t, cmd, handled)
	cmd, handled = m.handleResizeMouse(tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      startX,
		Y:      moveY,
	})
	runHandledCmd(t, cmd, handled)

	after := sessionSnapshot(t, m, snap.Name)
	beforePane := findSnapshotPane(t, snap, edge.PaneID)
	afterPane := findSnapshotPane(t, after, edge.PaneID)
	if beforePane.Height == afterPane.Height {
		t.Fatalf("expected height change after horizontal drag")
	}
}

func TestResizeModeNudgeIntegration(t *testing.T) {
	m := newTestModel(t)
	snap := startSessionWithGrid(t, m, "resize-key", "1x2")
	refreshModel(t, m)
	selectSessionForTest(t, m, snap.Name)
	m.enterResizeMode()
	m.resize.snap = false
	m.resize.key.snapState = sessiond.SnapState{}

	edge, ok := m.activeResizeEdge()
	if !ok {
		t.Fatalf("expected active resize edge")
	}
	key := tea.KeyRight
	if edge.Edge == sessiond.ResizeEdgeUp || edge.Edge == sessiond.ResizeEdgeDown {
		key = tea.KeyDown
	}
	cmd, handled := m.handleResizeModeKey(tea.KeyMsg{Type: key})
	runHandledCmd(t, cmd, handled)

	after := sessionSnapshot(t, m, snap.Name)
	beforePane := findSnapshotPane(t, snap, edge.PaneID)
	afterPane := findSnapshotPane(t, after, edge.PaneID)
	if edge.Edge == sessiond.ResizeEdgeUp || edge.Edge == sessiond.ResizeEdgeDown {
		if beforePane.Height == afterPane.Height {
			t.Fatalf("expected height change after keyboard nudge")
		}
		return
	}
	if beforePane.Width == afterPane.Width {
		t.Fatalf("expected width change after keyboard nudge")
	}
}

func TestContextMenuSplitIntegration(t *testing.T) {
	m := newTestModel(t)
	startSessionWithGrid(t, m, "context-menu", "1x1")
	refreshModel(t, m)
	selectSessionForTest(t, m, "context-menu")

	hit := firstPaneHit(t, m)
	cmd, handled := m.handleContextMenuMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonRight,
		X:      hit.Outer.X + 1,
		Y:      hit.Outer.Y + 1,
	})
	runHandledCmd(t, cmd, handled)
	if !m.contextMenu.open {
		t.Fatalf("expected context menu open")
	}
	index := findContextMenuIndex(m.contextMenu.items, contextMenuSplitRight)
	if index < 0 {
		t.Fatalf("context menu missing split right")
	}
	m.contextMenu.index = index
	runCmd(t, m.applyContextMenu())

	afterSplit := sessionSnapshot(t, m, "context-menu")
	if len(afterSplit.Panes) != 2 {
		t.Fatalf("expected 2 panes after split, got %d", len(afterSplit.Panes))
	}
}

func refreshModel(t *testing.T, m *Model) {
	t.Helper()
	cmd := m.requestRefreshCmd()
	if cmd == nil {
		t.Fatalf("refresh cmd nil")
	}
	msg := cmd()
	if msg == nil {
		t.Fatalf("refresh msg nil")
	}
	_, _ = m.Update(msg)
}

func startSessionWithGrid(t *testing.T, m *Model, name, grid string) native.SessionSnapshot {
	t.Helper()
	if m == nil || m.client == nil {
		t.Fatalf("session client unavailable")
	}
	if name == "" {
		name = "sess"
	}
	path := t.TempDir()
	writeProjectLayoutGrid(t, path, name, grid)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := m.client.StartSession(ctx, sessiond.StartSessionRequest{
		Name:       name,
		Path:       path,
		LayoutName: "",
	}); err != nil {
		t.Fatalf("StartSession() error: %v", err)
	}
	return waitForSessionSnapshot(t, m.client, name)
}

func writeProjectLayoutGrid(t *testing.T, path, session, grid string) {
	t.Helper()
	layoutCfg := &layout.LayoutConfig{Grid: grid}
	layoutYAML, err := layoutCfg.ToYAML()
	if err != nil {
		t.Fatalf("layout ToYAML error: %v", err)
	}
	content := fmt.Sprintf("session: %s\n\nlayout:\n", session)
	for _, line := range strings.Split(layoutYAML, "\n") {
		if line != "" {
			content += "  " + line + "\n"
		}
	}
	if err := os.WriteFile(filepath.Join(path, ".peakypanes.yml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .peakypanes.yml: %v", err)
	}
}

func selectSessionForTest(t *testing.T, m *Model, name string) {
	t.Helper()
	project, session := findProjectForSession(m.data.Projects, name)
	if project == nil || session == nil {
		t.Fatalf("session %q not found in model data", name)
	}
	paneIndex := session.ActivePane
	if paneIndex == "" && len(session.Panes) > 0 {
		paneIndex = session.Panes[0].Index
	}
	m.tab = TabProject
	m.applySelection(selectionState{ProjectID: project.ID, Session: session.Name, Pane: paneIndex})
	m.selectionVersion++
}

func firstPaneHit(t *testing.T, m *Model) mouse.PaneHit {
	t.Helper()
	hits := m.paneHits()
	if len(hits) == 0 {
		t.Fatalf("expected pane hit")
	}
	return hits[0]
}

func sessionSnapshot(t *testing.T, m *Model, name string) native.SessionSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := m.client.SnapshotState(ctx, 0)
	if err != nil {
		t.Fatalf("SnapshotState() error: %v", err)
	}
	for _, snap := range resp.Sessions {
		if snap.Name == name {
			return snap
		}
	}
	t.Fatalf("session %q not found in snapshot", name)
	return native.SessionSnapshot{}
}

func findSnapshotPane(t *testing.T, snap native.SessionSnapshot, paneID string) *native.PaneSnapshot {
	t.Helper()
	for i := range snap.Panes {
		if snap.Panes[i].ID == paneID {
			return &snap.Panes[i]
		}
	}
	t.Fatalf("pane %q not found in snapshot", paneID)
	return nil
}

func findContextMenuIndex(items []contextMenuItem, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func runHandledCmd(t *testing.T, cmd tea.Cmd, handled bool) {
	t.Helper()
	if !handled || cmd == nil {
		return
	}
	msg := cmd()
	checkCmdMsg(t, msg)
}

func runCmd(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	checkCmdMsg(t, msg)
}

func checkCmdMsg(t *testing.T, msg tea.Msg) {
	t.Helper()
	switch m := msg.(type) {
	case nil:
		return
	case ErrorMsg:
		if m.Err != nil {
			t.Fatalf("command error: %v", m.Err)
		}
	case tea.BatchMsg:
		for _, cmd := range m {
			if cmd == nil {
				continue
			}
			checkCmdMsg(t, cmd())
		}
	}
}
