package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
)

func TestNextDaemonReconnectDelay(t *testing.T) {
	if got := nextDaemonReconnectDelay(0); got != daemonReconnectMinDelay {
		t.Fatalf("got=%v", got)
	}
	if got := nextDaemonReconnectDelay(daemonReconnectMaxDelay); got != daemonReconnectMaxDelay {
		t.Fatalf("got=%v", got)
	}
}

func TestPaneClosedHelpers(t *testing.T) {
	if isPaneClosedError(nil) {
		t.Fatalf("nil should be false")
	}
	if !isPaneClosedError(errors.New("pane closed")) {
		t.Fatalf("expected true")
	}
	if paneClosedMessage(errors.New("pane closed")) != "Pane closed" {
		t.Fatalf("expected capitalized")
	}
	if paneClosedMessage(errors.New("Pane not found")) != "Pane closed" {
		t.Fatalf("expected pane closed")
	}
	msg := newPaneClosedMsg("p-1", errors.New("pane not found"))
	if msg.PaneID != "p-1" || msg.Message == "" {
		t.Fatalf("msg=%#v", msg)
	}
	if capitalizeFirst("x") != "X" {
		t.Fatalf("capitalizeFirst")
	}
}

func TestCornerCursorShape(t *testing.T) {
	ref := layoutgeom.CornerRef{
		Vertical:   layoutgeom.EdgeRef{Edge: sessiond.ResizeEdgeLeft},
		Horizontal: layoutgeom.EdgeRef{Edge: sessiond.ResizeEdgeUp},
	}
	if got := cornerCursorShape(ref); got != cursorShapeDiagNWSE {
		t.Fatalf("got=%v", got)
	}
}

func TestHandleCursorShapeFlushEmits(t *testing.T) {
	var emitted []string
	now := time.Now().UTC()
	m := &Model{
		state:                 StateDashboard,
		cursorShape:           cursorShapePointer,
		cursorShapePending:    cursorShapeRowResize,
		cursorShapeLastSentAt: now.Add(-2 * cursorShapeThrottle),
		oscEmit: func(seq string) {
			emitted = append(emitted, seq)
		},
	}
	_ = m.handleCursorShapeFlush(cursorShapeFlushMsg{At: now})
	if m.cursorShape != cursorShapeRowResize {
		t.Fatalf("shape=%v", m.cursorShape)
	}
	if len(emitted) != 1 || !strings.Contains(emitted[0], "ns-resize") {
		t.Fatalf("emitted=%v", emitted)
	}
}

func TestContextMenuCloseAndMove(t *testing.T) {
	m := &Model{}
	m.contextMenu = contextMenuState{
		open: true,
		items: []contextMenuItem{
			{ID: contextMenuClose, Enabled: false},
			{ID: contextMenuZoom, Enabled: true},
		},
		index: 0,
	}
	m.contextMenuMove(1)
	if m.contextMenu.index != 1 {
		t.Fatalf("index=%d", m.contextMenu.index)
	}
	m.closeContextMenu()
	if m.contextMenu.open {
		t.Fatalf("expected closed")
	}
}

func TestApplyContextMenuClosesOnInvalidIndex(t *testing.T) {
	m := &Model{}
	m.contextMenu = contextMenuState{
		open:  true,
		items: []contextMenuItem{{ID: contextMenuZoom, Enabled: true}},
		index: 99,
	}
	if cmd := m.applyContextMenu(); cmd != nil {
		t.Fatalf("expected nil cmd")
	}
	if m.contextMenu.open {
		t.Fatalf("expected menu closed")
	}
}

func TestHandleDaemonDisconnectMarksDisconnected(t *testing.T) {
	m := &Model{daemonVersion: "test"}
	if cmd := m.handleDaemonDisconnect(sessiond.ErrConnectionUnavailable); cmd == nil {
		t.Fatalf("expected reconnect cmd")
	}
	if !m.daemonDisconnected {
		t.Fatalf("expected disconnected")
	}
}
