//go:build integration

package sessiond

import (
	"context"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/tool"
)

func TestPaneViewResizeUpdatesWindow(t *testing.T) {
	mgr, err := native.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	reg, err := tool.DefaultRegistry()
	if err != nil {
		mgr.Close()
		t.Fatalf("DefaultRegistry: %v", err)
	}
	if err := mgr.SetToolRegistry(reg); err != nil {
		mgr.Close()
		t.Fatalf("SetToolRegistry: %v", err)
	}
	defer mgr.Close()

	layoutCfg := &layout.LayoutConfig{Grid: "1x2"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := mgr.StartSession(ctx, native.SessionSpec{
		Name:   "sess",
		Path:   t.TempDir(),
		Layout: layoutCfg,
	})
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if len(session.Panes) == 0 {
		t.Fatalf("expected panes in session")
	}
	paneID := session.Panes[0].ID
	win := mgr.Window(paneID)
	if win == nil {
		t.Fatalf("expected window")
	}

	d := &Daemon{manager: wrapManager(mgr), version: "test"}
	_, err = d.paneViewResponse(ctx, nil, paneID, PaneViewRequest{PaneID: paneID, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("paneViewResponse: %v", err)
	}
	if win.Cols() != 80 || win.Rows() != 24 {
		t.Fatalf("expected window 80x24, got %dx%d", win.Cols(), win.Rows())
	}

	_, err = d.paneViewResponse(ctx, nil, paneID, PaneViewRequest{PaneID: paneID, Cols: 100, Rows: 30})
	if err != nil {
		t.Fatalf("paneViewResponse resize: %v", err)
	}
	if win.Cols() != 100 || win.Rows() != 30 {
		t.Fatalf("expected window 100x30, got %dx%d", win.Cols(), win.Rows())
	}
}
