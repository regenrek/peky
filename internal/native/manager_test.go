package native

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/terminal"
)

func drainEvents(ch <-chan PaneEvent) int {
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			return count
		}
	}
}

func TestSessionNamesAndRenameSession(t *testing.T) {
	m := NewManager()
	m.sessions["alpha"] = &Session{Name: "alpha"}
	m.sessions["beta"] = &Session{Name: "beta"}

	names := m.SessionNames()
	if len(names) != 2 {
		t.Fatalf("SessionNames() len=%d want 2", len(names))
	}

	if err := m.RenameSession("missing", "new"); err == nil {
		t.Fatalf("RenameSession() should fail on missing session")
	}
	if err := m.RenameSession("alpha", "beta"); err == nil {
		t.Fatalf("RenameSession() should fail on duplicate name")
	}
	if err := m.RenameSession("alpha", "gamma"); err != nil {
		t.Fatalf("RenameSession() error: %v", err)
	}
	if m.sessions["alpha"] != nil || m.sessions["gamma"] == nil {
		t.Fatalf("RenameSession() did not move session")
	}
	if m.Version() == 0 {
		t.Fatalf("RenameSession() should increment version")
	}
}

func TestKillSessionRemovesPanes(t *testing.T) {
	m := NewManager()
	paneA := &Pane{ID: "p-1", Index: "0"}
	paneB := &Pane{ID: "p-2", Index: "1"}
	session := &Session{Name: "sess", Panes: []*Pane{paneA, paneB}}
	m.sessions["sess"] = session
	m.panes[paneA.ID] = paneA
	m.panes[paneB.ID] = paneB

	if err := m.KillSession("sess"); err != nil {
		t.Fatalf("KillSession() error: %v", err)
	}
	if _, ok := m.sessions["sess"]; ok {
		t.Fatalf("KillSession() should remove session")
	}
	if len(m.panes) != 0 {
		t.Fatalf("KillSession() should remove panes")
	}
	if m.Version() == 0 {
		t.Fatalf("KillSession() should increment version")
	}
	if got := drainEvents(m.Events()); got != 2 {
		t.Fatalf("KillSession() events=%d want 2", got)
	}
}

func TestRenamePaneUpdatesTitle(t *testing.T) {
	m := NewManager()
	pane := &Pane{ID: "p-1", Index: "0", Title: "old"}
	session := &Session{Name: "sess", Panes: []*Pane{pane}}
	m.sessions["sess"] = session

	if err := m.RenamePane("missing", "0", "new"); err == nil {
		t.Fatalf("RenamePane() should fail on missing session")
	}
	if err := m.RenamePane("sess", "missing", "new"); err == nil {
		t.Fatalf("RenamePane() should fail on missing pane")
	}
	if err := m.RenamePane("sess", "0", "new"); err != nil {
		t.Fatalf("RenamePane() error: %v", err)
	}
	if pane.Title != "new" {
		t.Fatalf("RenamePane() title=%q want %q", pane.Title, "new")
	}
}

func TestClosePaneRetilesAndActivates(t *testing.T) {
	m := NewManager()
	paneA := &Pane{ID: "p-1", Index: "0", Active: false, Width: 10, Height: 10}
	paneB := &Pane{ID: "p-2", Index: "1", Active: false, Width: 10, Height: 10}
	session := &Session{Name: "sess", Panes: []*Pane{paneA, paneB}}
	m.sessions["sess"] = session
	m.panes[paneA.ID] = paneA
	m.panes[paneB.ID] = paneB

	if err := m.ClosePane(context.Background(), "sess", "0"); err != nil {
		t.Fatalf("ClosePane() error: %v", err)
	}
	if len(session.Panes) != 1 {
		t.Fatalf("ClosePane() panes=%d want 1", len(session.Panes))
	}
	if !session.Panes[0].Active {
		t.Fatalf("ClosePane() should activate remaining pane")
	}
	if session.Panes[0].Width != layoutBaseSize || session.Panes[0].Height != layoutBaseSize {
		t.Fatalf("ClosePane() did not retile to full size")
	}
	if got := drainEvents(m.Events()); got < 2 {
		t.Fatalf("ClosePane() events=%d want >=2", got)
	}
}

func TestSwapPanesSwapsGeometry(t *testing.T) {
	m := NewManager()
	paneA := &Pane{ID: "p-1", Index: "0", Left: 0, Top: 0, Width: 100, Height: 100}
	paneB := &Pane{ID: "p-2", Index: "1", Left: 100, Top: 0, Width: 100, Height: 100}
	session := &Session{Name: "sess", Panes: []*Pane{paneA, paneB}}
	m.sessions["sess"] = session

	if err := m.SwapPanes("sess", "0", "1"); err != nil {
		t.Fatalf("SwapPanes() error: %v", err)
	}
	if paneA.Index != "1" || paneB.Index != "0" {
		t.Fatalf("SwapPanes() indexes=%q/%q", paneA.Index, paneB.Index)
	}
	if paneA.Left != 100 || paneB.Left != 0 {
		t.Fatalf("SwapPanes() did not swap geometry")
	}
}

func TestSendInputMouseErrors(t *testing.T) {
	m := NewManager()
	if err := m.SendInput("missing", []byte("hi")); err == nil {
		t.Fatalf("SendInput() should fail on missing pane")
	}
	if err := m.SendMouse("missing", nil); err == nil {
		t.Fatalf("SendMouse() should fail on missing pane")
	}

	pane := &Pane{ID: "p-1", Index: "0"}
	m.panes[pane.ID] = pane
	if err := m.SendInput("p-1", []byte("hi")); err == nil {
		t.Fatalf("SendInput() should fail without window")
	}
}

func TestSplitPaneErrors(t *testing.T) {
	m := NewManager()
	if _, err := m.SplitPane(context.Background(), "missing", "0", true, 50); err == nil {
		t.Fatalf("SplitPane() should fail on missing session")
	}

	tmp := t.TempDir()
	file := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	pane := &Pane{ID: "p-1", Index: "0"}
	session := &Session{Name: "sess", Path: file, Panes: []*Pane{pane}}
	m.sessions["sess"] = session
	if _, err := m.SplitPane(context.Background(), "sess", "0", true, 50); err == nil {
		t.Fatalf("SplitPane() should fail on invalid path")
	}
	if _, err := m.SplitPane(context.Background(), "sess", "missing", true, 50); err == nil {
		t.Fatalf("SplitPane() should fail on missing pane")
	}
}

func TestBuildPanesErrors(t *testing.T) {
	m := NewManager()
	if _, err := m.buildPanes(context.Background(), SessionSpec{}); err == nil {
		t.Fatalf("buildPanes() should fail on nil layout")
	}

	layoutCfg := &layout.LayoutConfig{}
	if _, err := m.buildPanes(context.Background(), SessionSpec{Layout: layoutCfg}); err == nil {
		t.Fatalf("buildPanes() should fail on empty layout")
	}

	layoutCfg = &layout.LayoutConfig{Grid: "not-a-grid"}
	if _, err := m.buildPanes(context.Background(), SessionSpec{Layout: layoutCfg}); err == nil {
		t.Fatalf("buildPanes() should fail on invalid grid")
	}
}

func TestBuildSplitPanesCanceledContext(t *testing.T) {
	m := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	defs := []layout.PaneDef{{Title: "one", Cmd: "echo hi"}}
	if _, err := m.buildSplitPanes(ctx, "", defs, nil); err == nil {
		t.Fatalf("buildSplitPanes() should fail when context cancelled")
	}
}

func TestNextPaneIDAndWindow(t *testing.T) {
	m := NewManager()
	if got := m.nextPaneID(); got != "p-1" {
		t.Fatalf("nextPaneID() = %q", got)
	}
	if got := m.nextPaneID(); got != "p-2" {
		t.Fatalf("nextPaneID() = %q", got)
	}

	pane := &Pane{ID: "p-1", window: &terminal.Window{}}
	m.panes[pane.ID] = pane
	if got := m.PaneWindow("p-1"); got == nil {
		t.Fatalf("PaneWindow() returned nil")
	}
}

func TestMarkActiveAndForwardUpdates(t *testing.T) {
	m := NewManager()
	m.forwardUpdates(nil)
	m.forwardUpdates(&Pane{ID: "p-1"})

	pane := &Pane{ID: "p-2"}
	m.panes[pane.ID] = pane
	m.markActive(pane.ID)
	if pane.LastActive.IsZero() {
		t.Fatalf("markActive() did not set LastActive")
	}
	if m.Version() == 0 {
		t.Fatalf("markActive() should increment version")
	}
	if got := drainEvents(m.Events()); got != 1 {
		t.Fatalf("markActive() events=%d want 1", got)
	}
}

func TestStartSessionLayoutNoPanes(t *testing.T) {
	m := NewManager()
	layoutCfg := &layout.LayoutConfig{Name: "empty"}
	_, err := m.StartSession(context.Background(), SessionSpec{
		Name:   "demo",
		Path:   t.TempDir(),
		Layout: layoutCfg,
	})
	if err == nil {
		t.Fatalf("expected error for empty layout")
	}
	if !strings.Contains(err.Error(), "layout has no panes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSplitCommandAndParsePercent(t *testing.T) {
	cmd, args, err := splitCommand("echo hi")
	if err != nil {
		t.Fatalf("splitCommand() error: %v", err)
	}
	if cmd != "echo" || !reflect.DeepEqual(args, []string{"hi"}) {
		t.Fatalf("splitCommand() = %q %#v", cmd, args)
	}
	if _, _, err := splitCommand(""); err == nil {
		t.Fatalf("splitCommand() should fail on empty")
	}

	if parsePercent("50%") != 50 {
		t.Fatalf("parsePercent(50%%) failed")
	}
	if parsePercent("0") != 0 {
		t.Fatalf("parsePercent(0) failed")
	}
	if parsePercent("bad") != 0 {
		t.Fatalf("parsePercent(bad) should be 0")
	}
}

func TestSplitRect(t *testing.T) {
	base := rect{x: 0, y: 0, w: 100, h: 40}
	left, right := splitRect(base, false, 25)
	if left.w != 75 || right.w != 25 {
		t.Fatalf("splitRect horiz widths=%d/%d", left.w, right.w)
	}
	top, bottom := splitRect(base, true, 50)
	if top.h != 20 || bottom.h != 20 {
		t.Fatalf("splitRect vert heights=%d/%d", top.h, bottom.h)
	}
}

func TestSnapshotWithNilWindow(t *testing.T) {
	m := NewManager()
	pane := &Pane{ID: "p-1", Index: "0", Title: "t", Command: "cmd"}
	session := &Session{Name: "sess", Panes: []*Pane{pane}}
	m.sessions["sess"] = session

	snaps := m.Snapshot(context.Background(), 3)
	if len(snaps) != 1 || len(snaps[0].Panes) != 1 {
		t.Fatalf("Snapshot() length mismatch")
	}
	if snaps[0].Panes[0].Preview != nil {
		t.Fatalf("Snapshot() preview should be nil without window")
	}
}

func TestManagerHelpers(t *testing.T) {
	var nilManager *Manager
	if nilManager.Session("any") != nil {
		t.Fatalf("Session() on nil manager should return nil")
	}
	if nilManager.Window("any") != nil {
		t.Fatalf("Window() on nil manager should return nil")
	}

	dir := t.TempDir()
	if err := validatePath(dir); err != nil {
		t.Fatalf("validatePath() error: %v", err)
	}
	if err := validatePath(""); err != nil {
		t.Fatalf("validatePath(\"\") error: %v", err)
	}

	panes := []*Pane{{Index: "2"}, {Index: "1"}, {Index: "A", Active: true}}
	if got := nextPaneIndex(panes); got != "3" {
		t.Fatalf("nextPaneIndex()=%q want 3", got)
	}
	if findPaneByIndex(panes, "1") == nil {
		t.Fatalf("findPaneByIndex() should find pane 1")
	}
	if !anyPaneActive(panes) {
		t.Fatalf("anyPaneActive() should be true")
	}
	sortPanesByIndex(panes)
	if panes[0].Index != "1" || panes[1].Index != "2" {
		t.Fatalf("sortPanesByIndex() unexpected order: %#v", panes)
	}
}

func TestManagerCloseClosesEvents(t *testing.T) {
	m := NewManager()
	m.Close()
	select {
	case _, ok := <-m.Events():
		if ok {
			t.Fatalf("Events() should be closed after Close")
		}
	default:
		t.Fatalf("Events() channel not closed")
	}
}
