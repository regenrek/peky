package sessiond

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/termframe"
)

type stubPaneView struct {
	frameCalled bool
	frameDirect bool
}

func (s *stubPaneView) UpdateSeq() uint64 {
	return 0
}

func (s *stubPaneView) FrameCacheSeq() uint64 {
	return 0
}

func (s *stubPaneView) CopyModeActive() bool {
	return false
}

func (s *stubPaneView) ViewFrameCtx(ctx context.Context) (termframe.Frame, error) {
	s.frameCalled = true
	return termframe.Frame{Cols: 1, Rows: 1, Cells: []termframe.Cell{{Content: "A", Width: 1}}}, nil
}

func (s *stubPaneView) ViewFrameDirectCtx(ctx context.Context) (termframe.Frame, error) {
	s.frameDirect = true
	return termframe.Frame{Cols: 1, Rows: 1, Cells: []termframe.Cell{{Content: "B", Width: 1}}}, nil
}

func TestRequirePaneID(t *testing.T) {
	if _, err := requirePaneID(" "); err == nil {
		t.Fatalf("expected error for empty pane id")
	}
	got, err := requirePaneID(" pane-1 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pane-1" {
		t.Fatalf("expected trimmed pane id, got %q", got)
	}
}

func TestNormalizeDimensions(t *testing.T) {
	cols, rows := normalizeDimensions(0, 0)
	if cols != 1 || rows != 1 {
		t.Fatalf("expected 1x1, got %dx%d", cols, rows)
	}
	cols, rows = normalizeDimensions(5, -2)
	if cols != 5 || rows != 1 {
		t.Fatalf("expected 5x1, got %dx%d", cols, rows)
	}
	cols, rows = normalizeDimensions(limits.PaneMaxCols+10, limits.PaneMaxRows+10)
	if cols != limits.PaneMaxCols || rows != limits.PaneMaxRows {
		t.Fatalf("expected clamp to %dx%d, got %dx%d", limits.PaneMaxCols, limits.PaneMaxRows, cols, rows)
	}
}

func TestResolveResizeEdge(t *testing.T) {
	edge, err := resolveResizeEdge(ResizeEdgeLeft)
	if err != nil || edge != layout.ResizeEdgeLeft {
		t.Fatalf("expected left edge, got %v err=%v", edge, err)
	}
	if _, err := resolveResizeEdge(ResizeEdge("bad")); err == nil {
		t.Fatalf("expected error for invalid edge")
	}
}

func TestPaneViewFrame(t *testing.T) {
	win := &stubPaneView{}
	frame, err := paneViewFrame(context.Background(), win, PaneViewRequest{})
	if err != nil {
		t.Fatalf("paneViewFrame: %v", err)
	}
	if frame.Cols != 1 || frame.Rows != 1 || len(frame.Cells) != 1 || frame.Cells[0].Content != "A" {
		t.Fatalf("unexpected frame from cached render")
	}
	if !win.frameCalled || win.frameDirect {
		t.Fatalf("expected cached frame render")
	}

	win = &stubPaneView{}
	frame, err = paneViewFrame(context.Background(), win, PaneViewRequest{DirectRender: true})
	if err != nil {
		t.Fatalf("paneViewFrame: %v", err)
	}
	if frame.Cells[0].Content != "B" || !win.frameDirect {
		t.Fatalf("expected direct frame render")
	}
}

func TestHandleRequestError(t *testing.T) {
	d := &Daemon{}
	resp := d.handleRequest(Envelope{Kind: EnvelopeRequest, Op: Op("unknown"), ID: 7})
	if resp.Kind != EnvelopeResponse {
		t.Fatalf("expected response envelope kind")
	}
	if resp.Error == "" {
		t.Fatalf("expected error for unknown op")
	}
	if resp.ID != 7 {
		t.Fatalf("expected response id 7, got %d", resp.ID)
	}
}

func TestWindowFromRequestErrors(t *testing.T) {
	d := &Daemon{}
	if _, _, err := d.windowFromRequest(""); err == nil {
		t.Fatalf("expected error for empty pane id")
	}
	if _, _, err := d.windowFromRequest("pane"); err == nil {
		t.Fatalf("expected error for nil manager")
	}
	mgr, err := native.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	d.manager = wrapManager(mgr)
	t.Cleanup(func() { d.manager.Close() })
	if _, _, err := d.windowFromRequest("missing"); err == nil {
		t.Fatalf("expected error for missing pane")
	}
}

func TestStartSessionProjectLayoutNoPanes(t *testing.T) {
	dir := t.TempDir()
	config := []byte("layout:\n  name: empty\n  panes: []\n")
	if err := os.WriteFile(filepath.Join(dir, ".peky.yml"), config, 0o600); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	mgr, err := native.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	d := &Daemon{manager: wrapManager(mgr)}
	t.Cleanup(func() { d.manager.Close() })
	_, err = d.startSession(StartSessionRequest{Name: "demo", Path: dir})
	if err == nil {
		t.Fatalf("expected error for empty layout")
	}
	if !strings.Contains(err.Error(), "layout has no panes defined") {
		t.Fatalf("unexpected error: %v", err)
	}
}
