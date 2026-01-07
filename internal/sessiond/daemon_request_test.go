package sessiond

import (
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/tool"
)

func newTestDaemon(t *testing.T) *Daemon {
	t.Helper()
	mgr, err := native.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	reg, err := tool.DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	if err := mgr.SetToolRegistry(reg); err != nil {
		t.Fatalf("SetToolRegistry: %v", err)
	}
	d := &Daemon{manager: wrapManager(mgr), toolRegistry: reg, version: "test"}
	t.Cleanup(func() {
		if d.manager != nil {
			d.manager.Close()
		}
	})
	return d
}

func TestHandleRequestPayloadHello(t *testing.T) {
	d := newTestDaemon(t)
	payload, err := encodePayload(HelloRequest{Version: "client"})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	data, err := d.handleRequestPayload(Envelope{Op: OpHello, Payload: payload})
	if err != nil {
		t.Fatalf("handleRequestPayload: %v", err)
	}
	var resp HelloResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if resp.Version != d.version {
		t.Fatalf("expected version %q, got %q", d.version, resp.Version)
	}
	if resp.PID <= 0 {
		t.Fatalf("expected pid set")
	}
}

func TestHandleRequestPayloadSessionNames(t *testing.T) {
	d := newTestDaemon(t)
	data, err := d.handleRequestPayload(Envelope{Op: OpSessionNames})
	if err != nil {
		t.Fatalf("handleRequestPayload: %v", err)
	}
	var resp SessionNamesResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(resp.Names) != 0 {
		t.Fatalf("expected no sessions, got %#v", resp.Names)
	}
}

func TestHandleRequestPayloadSnapshot(t *testing.T) {
	d := newTestDaemon(t)
	payload, err := encodePayload(SnapshotRequest{PreviewLines: 2})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	data, err := d.handleRequestPayload(Envelope{Op: OpSnapshot, Payload: payload})
	if err != nil {
		t.Fatalf("handleRequestPayload: %v", err)
	}
	var resp SnapshotResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(resp.Sessions) != 0 {
		t.Fatalf("expected empty snapshot")
	}
}

func TestHandleRequestPayloadStartSessionInvalidPath(t *testing.T) {
	d := newTestDaemon(t)
	payload, err := encodePayload(StartSessionRequest{Path: filepath.Join(t.TempDir(), "missing")})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	if _, err := d.handleRequestPayload(Envelope{Op: OpStartSession, Payload: payload}); err == nil {
		t.Fatalf("expected error for invalid path")
	}
}

func TestHandleRequestPayloadResizePaneMissing(t *testing.T) {
	d := newTestDaemon(t)
	payload, err := encodePayload(ResizePaneRequest{SessionName: "sess", PaneID: "missing", Edge: ResizeEdgeLeft, Delta: 2, Snap: true})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	if _, err := d.handleRequestPayload(Envelope{Op: OpResizePane, Payload: payload}); err == nil {
		t.Fatalf("expected error for missing pane")
	}
}

func TestHandleRequestPayloadSendMouseInvalid(t *testing.T) {
	d := newTestDaemon(t)
	payload, err := encodePayload(SendMouseRequest{PaneID: "pane", Event: MouseEventPayload{X: -1, Y: 0}})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	data, err := d.handleRequestPayload(Envelope{Op: OpSendMouse, Payload: payload})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Fatalf("expected nil payload for invalid mouse event")
	}
}

func TestHandleRequestPayloadUnknownOp(t *testing.T) {
	d := newTestDaemon(t)
	if _, err := d.handleRequestPayload(Envelope{Op: Op("bogus")}); err == nil {
		t.Fatalf("expected error for unknown op")
	}
}
