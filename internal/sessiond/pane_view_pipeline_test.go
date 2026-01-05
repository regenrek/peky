package sessiond

import (
	"context"
	"testing"

	"github.com/regenrek/peakypanes/internal/termframe"
)

func TestPaneViewResponseCachesFrame(t *testing.T) {
	frame := termframe.Frame{Cols: 2, Rows: 1, Cells: []termframe.Cell{{Content: "A", Width: 1}, {Content: "B", Width: 1}}}
	win := &fakeTerminalWindow{viewFrame: frame, updateSeq: 1}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}
	client := &clientConn{paneViewCache: make(map[paneViewCacheKey]cachedPaneView)}

	req := PaneViewRequest{PaneID: "pane-1", Cols: 80, Rows: 24}
	resp, err := d.paneViewResponse(context.Background(), client, "pane-1", req)
	if err != nil {
		t.Fatalf("paneViewResponse error: %v", err)
	}
	if resp.Frame.Empty() || resp.UpdateSeq != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if win.calls["viewFrame"] != 1 {
		t.Fatalf("expected viewFrame called once, got %#v", win.calls)
	}

	resp2, err := d.paneViewResponse(context.Background(), client, "pane-1", req)
	if err != nil {
		t.Fatalf("paneViewResponse cached error: %v", err)
	}
	if resp2.UpdateSeq != 1 || resp2.Frame.Empty() {
		t.Fatalf("unexpected cached response: %#v", resp2)
	}
	if win.calls["viewFrame"] != 1 {
		t.Fatalf("expected cached response to skip render, got %#v", win.calls)
	}

	win.updateSeq = 2
	resp3, err := d.paneViewResponse(context.Background(), client, "pane-1", req)
	if err != nil {
		t.Fatalf("paneViewResponse after update error: %v", err)
	}
	if resp3.UpdateSeq != 2 || resp3.Frame.Empty() {
		t.Fatalf("unexpected updated response: %#v", resp3)
	}
	if win.calls["viewFrame"] != 2 {
		t.Fatalf("expected render after update, got %#v", win.calls)
	}
}

func TestPaneViewNotModifiedSkipsRender(t *testing.T) {
	win := &fakeTerminalWindow{viewFrame: termframe.Frame{Cols: 1, Rows: 1, Cells: []termframe.Cell{{Content: "x", Width: 1}}}, updateSeq: 7}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	req := PaneViewRequest{PaneID: "pane-1", Cols: 10, Rows: 3, KnownSeq: 7}
	resp, err := d.paneViewResponse(context.Background(), nil, "pane-1", req)
	if err != nil {
		t.Fatalf("paneViewResponse error: %v", err)
	}
	if !resp.NotModified || resp.UpdateSeq != 7 {
		t.Fatalf("expected not-modified response, got %#v", resp)
	}
	if win.calls["viewFrame"] != 0 {
		t.Fatalf("expected no render for not-modified, got %#v", win.calls)
	}
}

func TestPaneViewDirectUsesDirectFrame(t *testing.T) {
	win := &fakeTerminalWindow{viewFrame: termframe.Frame{Cols: 1, Rows: 1, Cells: []termframe.Cell{{Content: "x", Width: 1}}}, updateSeq: 1}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	req := PaneViewRequest{PaneID: "pane-1", Cols: 10, Rows: 3, DirectRender: true}
	resp, err := d.paneViewResponse(context.Background(), nil, "pane-1", req)
	if err != nil {
		t.Fatalf("paneViewResponse error: %v", err)
	}
	if resp.Frame.Empty() || resp.UpdateSeq != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if win.calls["viewFrameDirect"] != 1 {
		t.Fatalf("expected direct render, got %#v", win.calls)
	}
	if win.calls["viewFrame"] != 0 {
		t.Fatalf("expected direct render to skip cached render, got %#v", win.calls)
	}
}
