package sessiond

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/native"
)

type stubPaneView struct {
	lipglossCalled bool
	ansiCalled     bool
	ansiDirect     bool
	showCursor     bool
	profile        termenv.Profile
}

func (s *stubPaneView) UpdateSeq() uint64 {
	return 0
}

func (s *stubPaneView) ANSICacheSeq() uint64 {
	return 0
}

func (s *stubPaneView) CopyModeActive() bool {
	return false
}

func (s *stubPaneView) ViewLipglossCtx(ctx context.Context, showCursor bool, profile termenv.Profile) (string, error) {
	s.lipglossCalled = true
	s.showCursor = showCursor
	s.profile = profile
	return "lipgloss", nil
}

func (s *stubPaneView) ViewANSICtx(ctx context.Context) (string, error) {
	s.ansiCalled = true
	return "ansi", nil
}

func (s *stubPaneView) ViewANSIDirectCtx(ctx context.Context) (string, error) {
	s.ansiDirect = true
	return "ansi", nil
}

func (s *stubPaneView) ViewLipgloss(showCursor bool, profile termenv.Profile) string {
	s.lipglossCalled = true
	s.showCursor = showCursor
	s.profile = profile
	return "lipgloss"
}

func (s *stubPaneView) ViewANSI() string {
	s.ansiCalled = true
	return "ansi"
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

func TestPaneViewString(t *testing.T) {
	win := &stubPaneView{}
	out, err := paneViewString(context.Background(), win, PaneViewRequest{Mode: PaneViewLipgloss, ShowCursor: true, ColorProfile: termenv.ANSI256})
	if err != nil {
		t.Fatalf("paneViewString: %v", err)
	}
	if out != "lipgloss" || !win.lipglossCalled || !win.showCursor || win.profile != termenv.ANSI256 {
		t.Fatalf("expected lipgloss render")
	}

	win = &stubPaneView{}
	out, err = paneViewString(context.Background(), win, PaneViewRequest{Mode: PaneViewANSI})
	if err != nil {
		t.Fatalf("paneViewString: %v", err)
	}
	if out != "ansi" || !win.ansiCalled {
		t.Fatalf("expected ansi render")
	}

	win = &stubPaneView{}
	out, err = paneViewString(context.Background(), win, PaneViewRequest{Mode: PaneViewANSI, DirectRender: true})
	if err != nil {
		t.Fatalf("paneViewString: %v", err)
	}
	if out != "ansi" || !win.ansiDirect {
		t.Fatalf("expected direct ansi render")
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
	if err := os.WriteFile(filepath.Join(dir, ".peakypanes.yml"), config, 0o600); err != nil {
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
