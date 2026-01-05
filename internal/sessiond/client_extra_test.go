package sessiond

import (
	"context"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/termframe"
)

func newClientWithTestServer(t *testing.T) (*Client, func()) {
	t.Helper()
	serverConn, clientConn := net.Pipe()

	client := &Client{
		conn:    clientConn,
		pending: make(map[uint64]chan Envelope),
		events:  make(chan Event, 4),
		version: "test",
	}
	go client.readLoop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		serveClientTestConn(t, serverConn)
	}()

	cleanup := func() {
		_ = client.Close()
		_ = serverConn.Close()
		_ = clientConn.Close()
		<-done
	}
	return client, cleanup
}

func TestClientHelloAndEvents(t *testing.T) {
	client, cleanup := newClientWithTestServer(t)
	defer cleanup()

	ctx := context.Background()
	if err := client.hello(ctx); err != nil {
		t.Fatalf("hello error: %v", err)
	}
	evt := waitForEvent(t, client.Events())
	if evt.Type != EventSessionChanged || evt.Session != "alpha" {
		t.Fatalf("unexpected event: %#v", evt)
	}
}

func TestClientSessionQueries(t *testing.T) {
	client, cleanup := newClientWithTestServer(t)
	defer cleanup()

	ctx := context.Background()
	names, err := client.SessionNames(ctx)
	if err != nil || len(names) != 2 || names[0] != "alpha" {
		t.Fatalf("SessionNames: %#v err=%v", names, err)
	}

	sessions, version, err := client.Snapshot(ctx, 2)
	if err != nil || version != 42 || len(sessions) != 1 || sessions[0].Name != "alpha" {
		t.Fatalf("Snapshot: sessions=%#v version=%d err=%v", sessions, version, err)
	}
}

func TestClientSessionMutations(t *testing.T) {
	client, cleanup := newClientWithTestServer(t)
	defer cleanup()

	ctx := context.Background()
	startResp, err := client.StartSession(ctx, StartSessionRequest{Name: "alpha", Path: "/tmp"})
	if err != nil || startResp.Name != "alpha" {
		t.Fatalf("StartSession: %#v err=%v", startResp, err)
	}

	renameResp, err := client.RenameSession(ctx, "alpha", "beta")
	if err != nil || renameResp.NewName != "beta" {
		t.Fatalf("RenameSession: %#v err=%v", renameResp, err)
	}

	if err := client.RenamePane(ctx, "alpha", "1", "new"); err != nil {
		t.Fatalf("RenamePane: %v", err)
	}

	if _, err := client.SplitPane(ctx, "alpha", "1", true, 50); err != nil {
		t.Fatalf("SplitPane: %v", err)
	}

	if err := client.ClosePane(ctx, "alpha", "1"); err != nil {
		t.Fatalf("ClosePane: %v", err)
	}
	if err := client.SwapPanes(ctx, "alpha", "1", "2"); err != nil {
		t.Fatalf("SwapPanes: %v", err)
	}
}

func TestClientPaneOps(t *testing.T) {
	client, cleanup := newClientWithTestServer(t)
	defer cleanup()

	ctx := context.Background()
	if err := client.SendInput(ctx, "pane-1", []byte("hi")); err != nil {
		t.Fatalf("SendInput: %v", err)
	}
	if err := client.SendMouse(ctx, "pane-1", MouseEventPayload{X: 1}); err != nil {
		t.Fatalf("SendMouse: %v", err)
	}
	if err := client.ResizePane(ctx, "pane-1", 80, 24); err != nil {
		t.Fatalf("ResizePane: %v", err)
	}
	if _, err := client.GetPaneView(ctx, PaneViewRequest{PaneID: "pane-1", Cols: 10, Rows: 2}); err != nil {
		t.Fatalf("GetPaneView: %v", err)
	}
	if _, err := client.TerminalAction(ctx, TerminalActionRequest{PaneID: "pane-1", Action: TerminalScrollUp}); err != nil {
		t.Fatalf("TerminalAction: %v", err)
	}
	if _, err := client.HandleTerminalKey(ctx, TerminalKeyRequest{PaneID: "pane-1", Key: "j"}); err != nil {
		t.Fatalf("HandleTerminalKey: %v", err)
	}
}

func TestClientSendNoConn(t *testing.T) {
	c := &Client{}
	if err := c.send(context.Background(), Envelope{}); err == nil {
		t.Fatalf("expected send error")
	}
}

func TestClientCallNil(t *testing.T) {
	var c *Client
	if _, err := c.call(context.Background(), OpHello, nil, nil); err == nil {
		t.Fatalf("expected nil client error")
	}
}

func serveClientTestConn(t *testing.T, conn net.Conn) {
	t.Helper()
	for {
		env, err := readEnvelope(conn)
		if err != nil {
			return
		}
		if env.Kind != EnvelopeRequest {
			continue
		}
		var payload []byte
		switch env.Op {
		case OpHello:
			payload = encodeMust(t, HelloResponse{Version: "test", PID: 123})
		case OpSessionNames:
			payload = encodeMust(t, SessionNamesResponse{Names: []string{"alpha", "beta"}})
		case OpSnapshot:
			payload = encodeMust(t, SnapshotResponse{
				Version:  42,
				Sessions: []native.SessionSnapshot{{Name: "alpha"}},
			})
		case OpStartSession:
			payload = encodeMust(t, StartSessionResponse{Name: "alpha", Path: "/tmp", LayoutName: "dev"})
		case OpRenameSession:
			payload = encodeMust(t, RenameSessionResponse{NewName: "beta"})
		case OpSplitPane:
			payload = encodeMust(t, SplitPaneResponse{NewIndex: "2"})
		case OpPaneView:
			payload = encodeMust(t, PaneViewResponse{PaneID: "pane-1", Cols: 10, Rows: 2, Frame: termframe.Frame{Cols: 1, Rows: 1, Cells: []termframe.Cell{{Content: "ok", Width: 1}}}})
		case OpTerminalAction:
			payload = encodeMust(t, TerminalActionResponse{PaneID: "pane-1"})
		case OpHandleKey:
			payload = encodeMust(t, TerminalKeyResponse{Handled: true})
		default:
			payload = nil
		}

		resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Payload: payload}
		if err := writeEnvelope(conn, resp); err != nil {
			return
		}
		if env.Op == OpHello {
			evtPayload := encodeMust(t, Event{Type: EventSessionChanged, Session: "alpha"})
			_ = writeEnvelope(conn, Envelope{Kind: EnvelopeEvent, Event: EventSessionChanged, Payload: evtPayload})
		}
	}
}

func encodeMust(t *testing.T, v any) []byte {
	t.Helper()
	data, err := encodePayload(v)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	return data
}

func waitForEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	deadline := time.After(250 * time.Millisecond)
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				t.Fatalf("events channel closed")
			}
			return evt
		case <-deadline:
			t.Fatalf("expected event")
		default:
			runtime.Gosched()
		}
	}
}
