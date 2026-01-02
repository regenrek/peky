package sessiond

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/native"
)

type clientCase struct {
	name    string
	op      Op
	check   func(Envelope) error
	respond any
	call    func(*Client) error
}

func runClientCase(t *testing.T, tc clientCase) {
	t.Helper()
	client, server := newTestClient(t)
	errCh := make(chan error, 1)

	go func() {
		env, err := readEnvelope(server)
		if err != nil {
			errCh <- err
			return
		}
		if env.Op != tc.op {
			errCh <- fmt.Errorf("expected op %q, got %q", tc.op, env.Op)
			return
		}
		if tc.check != nil {
			if err := tc.check(env); err != nil {
				errCh <- err
				return
			}
		}
		payload, err := encodePayload(tc.respond)
		if err != nil {
			errCh <- err
			return
		}
		resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Payload: payload}
		if err := writeEnvelope(server, resp); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	if err := tc.call(client); err != nil {
		t.Fatalf("%s call failed: %v", tc.name, err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("%s server error: %v", tc.name, err)
	}
}

func TestClientSnapshot(t *testing.T) {
	runClientCase(t, clientCase{
		name: "Snapshot",
		op:   OpSnapshot,
		check: func(env Envelope) error {
			var req SnapshotRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PreviewLines != 3 {
				return fmt.Errorf("expected preview lines 3, got %d", req.PreviewLines)
			}
			return nil
		},
		respond: SnapshotResponse{
			Version:  9,
			Sessions: []native.SessionSnapshot{{Name: "demo"}},
		},
		call: func(c *Client) error {
			sessions, version, err := c.Snapshot(context.Background(), 3)
			if err != nil {
				return err
			}
			if version != 9 || len(sessions) != 1 || sessions[0].Name != "demo" {
				return fmt.Errorf("unexpected snapshot response")
			}
			return nil
		},
	})
}

func TestClientStartSession(t *testing.T) {
	runClientCase(t, clientCase{
		name: "StartSession",
		op:   OpStartSession,
		check: func(env Envelope) error {
			var req StartSessionRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.Name != "demo" || req.Path != "/tmp/demo" {
				return fmt.Errorf("unexpected start request")
			}
			return nil
		},
		respond: StartSessionResponse{Name: "demo", Path: "/tmp/demo", LayoutName: "dev"},
		call: func(c *Client) error {
			resp, err := c.StartSession(context.Background(), StartSessionRequest{Name: "demo", Path: "/tmp/demo"})
			if err != nil {
				return err
			}
			if resp.Name != "demo" || resp.Path != "/tmp/demo" {
				return fmt.Errorf("unexpected start response")
			}
			return nil
		},
	})
}

func TestClientKillSession(t *testing.T) {
	runClientCase(t, clientCase{
		name: "KillSession",
		op:   OpKillSession,
		check: func(env Envelope) error {
			var req KillSessionRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.Name != "demo" {
				return fmt.Errorf("unexpected kill request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.KillSession(context.Background(), "demo")
		},
	})
}

func TestClientRenameSession(t *testing.T) {
	runClientCase(t, clientCase{
		name: "RenameSession",
		op:   OpRenameSession,
		check: func(env Envelope) error {
			var req RenameSessionRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.OldName != "old" || req.NewName != "new" {
				return fmt.Errorf("unexpected rename request")
			}
			return nil
		},
		respond: RenameSessionResponse{NewName: "new"},
		call: func(c *Client) error {
			resp, err := c.RenameSession(context.Background(), "old", "new")
			if err != nil {
				return err
			}
			if resp.NewName != "new" {
				return fmt.Errorf("unexpected rename response")
			}
			return nil
		},
	})
}

func TestClientRenamePane(t *testing.T) {
	runClientCase(t, clientCase{
		name: "RenamePane",
		op:   OpRenamePane,
		check: func(env Envelope) error {
			var req RenamePaneRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.SessionName != "session" || req.PaneIndex != "1" || req.NewTitle != "title" {
				return fmt.Errorf("unexpected rename pane request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.RenamePane(context.Background(), "session", "1", "title")
		},
	})
}

func TestClientSetPaneTool(t *testing.T) {
	runClientCase(t, clientCase{
		name: "SetPaneTool",
		op:   OpSetPaneTool,
		check: func(env Envelope) error {
			var req SetPaneToolRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PaneID != "pane-1" || req.Tool != "codex" {
				return fmt.Errorf("unexpected set tool request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.SetPaneTool(context.Background(), "pane-1", "codex")
		},
	})
}

func TestClientSplitPane(t *testing.T) {
	runClientCase(t, clientCase{
		name: "SplitPane",
		op:   OpSplitPane,
		check: func(env Envelope) error {
			var req SplitPaneRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.SessionName != "session" || req.PaneIndex != "1" || !req.Vertical || req.Percent != 40 {
				return fmt.Errorf("unexpected split request")
			}
			return nil
		},
		respond: SplitPaneResponse{NewIndex: "2"},
		call: func(c *Client) error {
			newIndex, err := c.SplitPane(context.Background(), "session", "1", true, 40)
			if err != nil {
				return err
			}
			if newIndex != "2" {
				return fmt.Errorf("unexpected split response")
			}
			return nil
		},
	})
}

func TestClientClosePane(t *testing.T) {
	runClientCase(t, clientCase{
		name: "ClosePane",
		op:   OpClosePane,
		check: func(env Envelope) error {
			var req ClosePaneRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.SessionName != "session" || req.PaneIndex != "1" {
				return fmt.Errorf("unexpected close request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.ClosePane(context.Background(), "session", "1")
		},
	})
}

func TestClientSwapPanes(t *testing.T) {
	runClientCase(t, clientCase{
		name: "SwapPanes",
		op:   OpSwapPanes,
		check: func(env Envelope) error {
			var req SwapPanesRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.SessionName != "session" || req.PaneA != "1" || req.PaneB != "2" {
				return fmt.Errorf("unexpected swap request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.SwapPanes(context.Background(), "session", "1", "2")
		},
	})
}

func TestClientSendInput(t *testing.T) {
	runClientCase(t, clientCase{
		name: "SendInput",
		op:   OpSendInput,
		check: func(env Envelope) error {
			var req SendInputRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PaneID != "pane" || string(req.Input) != "hi" {
				return fmt.Errorf("unexpected send input request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.SendInput(context.Background(), "pane", []byte("hi"))
		},
	})
}

func TestClientSendMouse(t *testing.T) {
	runClientCase(t, clientCase{
		name: "SendMouse",
		op:   OpSendMouse,
		check: func(env Envelope) error {
			var req SendMouseRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PaneID != "pane" || req.Event.X != 1 || req.Event.Action != MouseActionPress {
				return fmt.Errorf("unexpected send mouse request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.SendMouse(context.Background(), "pane", MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionPress})
		},
	})
}

func TestClientResizePane(t *testing.T) {
	runClientCase(t, clientCase{
		name: "ResizePane",
		op:   OpResizePane,
		check: func(env Envelope) error {
			var req ResizePaneRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PaneID != "pane" || req.Cols != 80 || req.Rows != 24 {
				return fmt.Errorf("unexpected resize request")
			}
			return nil
		},
		call: func(c *Client) error {
			return c.ResizePane(context.Background(), "pane", 80, 24)
		},
	})
}

func TestClientPaneView(t *testing.T) {
	runClientCase(t, clientCase{
		name: "GetPaneView",
		op:   OpPaneView,
		check: func(env Envelope) error {
			var req PaneViewRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PaneID != "pane" || req.Cols != 100 || req.Mode != PaneViewANSI {
				return fmt.Errorf("unexpected pane view request")
			}
			return nil
		},
		respond: PaneViewResponse{PaneID: "pane", View: "ok", Cols: 100, Rows: 40},
		call: func(c *Client) error {
			resp, err := c.GetPaneView(context.Background(), PaneViewRequest{PaneID: "pane", Cols: 100, Rows: 40})
			if err != nil {
				return err
			}
			if resp.View != "ok" || resp.PaneID != "pane" {
				return fmt.Errorf("unexpected pane view response")
			}
			return nil
		},
	})
}

func TestClientTerminalAction(t *testing.T) {
	runClientCase(t, clientCase{
		name: "TerminalAction",
		op:   OpTerminalAction,
		check: func(env Envelope) error {
			var req TerminalActionRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PaneID != "pane" || req.Action != TerminalPageDown {
				return fmt.Errorf("unexpected terminal action request")
			}
			return nil
		},
		respond: TerminalActionResponse{PaneID: "pane", Text: "done"},
		call: func(c *Client) error {
			resp, err := c.TerminalAction(context.Background(), TerminalActionRequest{PaneID: "pane", Action: TerminalPageDown})
			if err != nil {
				return err
			}
			if resp.Text != "done" {
				return fmt.Errorf("unexpected terminal action response")
			}
			return nil
		},
	})
}

func TestClientHandleTerminalKey(t *testing.T) {
	runClientCase(t, clientCase{
		name: "HandleTerminalKey",
		op:   OpHandleKey,
		check: func(env Envelope) error {
			var req TerminalKeyRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				return err
			}
			if req.PaneID != "pane" || req.Key != "esc" {
				return fmt.Errorf("unexpected terminal key request")
			}
			return nil
		},
		respond: TerminalKeyResponse{Handled: true, Toast: "ok"},
		call: func(c *Client) error {
			resp, err := c.HandleTerminalKey(context.Background(), TerminalKeyRequest{PaneID: "pane", Key: "esc"})
			if err != nil {
				return err
			}
			if !resp.Handled || resp.Toast != "ok" {
				return fmt.Errorf("unexpected terminal key response")
			}
			return nil
		},
	})
}

func serveHello(t *testing.T, ln net.Listener, expectedVersion string) <-chan error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		defer func() { _ = conn.Close() }()

		env, err := readEnvelope(conn)
		if err != nil {
			errCh <- err
			return
		}
		if env.Op != OpHello {
			errCh <- fmt.Errorf("expected hello op, got %q", env.Op)
			return
		}
		var req HelloRequest
		if err := decodePayload(env.Payload, &req); err != nil {
			errCh <- err
			return
		}
		if req.Version != expectedVersion {
			errCh <- fmt.Errorf("expected version %q, got %q", expectedVersion, req.Version)
			return
		}
		payload, err := encodePayload(HelloResponse{Version: expectedVersion, PID: 1})
		if err != nil {
			errCh <- err
			return
		}
		resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Payload: payload}
		if err := writeEnvelope(conn, resp); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	return errCh
}

func TestDialAndProbeDaemon(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "sessiond.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	errCh := serveHello(t, ln, "v1")
	client, err := Dial(context.Background(), socketPath, "v1")
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}

	ln2, err := net.Listen("unix", filepath.Join(dir, "probe.sock"))
	if err != nil {
		t.Fatalf("listen probe: %v", err)
	}
	t.Cleanup(func() { _ = ln2.Close() })
	errCh = serveHello(t, ln2, "v2")
	if err := probeDaemon(context.Background(), ln2.Addr().String(), "v2"); err != nil {
		t.Fatalf("probeDaemon: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("probe server error: %v", err)
	}
}

func TestClientClone(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "sessiond.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	errCh := make(chan error, 1)
	go func() {
		for i := 0; i < 2; i++ {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
				return
			}
			env, err := readEnvelope(conn)
			if err != nil {
				_ = conn.Close()
				errCh <- err
				return
			}
			if env.Op != OpHello {
				_ = conn.Close()
				errCh <- fmt.Errorf("expected hello op, got %q", env.Op)
				return
			}
			var req HelloRequest
			if err := decodePayload(env.Payload, &req); err != nil {
				_ = conn.Close()
				errCh <- err
				return
			}
			if req.Version != "v1" {
				_ = conn.Close()
				errCh <- fmt.Errorf("expected version %q, got %q", "v1", req.Version)
				return
			}
			payload, err := encodePayload(HelloResponse{Version: "v1", PID: 1})
			if err != nil {
				_ = conn.Close()
				errCh <- err
				return
			}
			resp := Envelope{Kind: EnvelopeResponse, Op: env.Op, ID: env.ID, Payload: payload}
			if err := writeEnvelope(conn, resp); err != nil {
				_ = conn.Close()
				errCh <- err
				return
			}
			_ = conn.Close()
		}
		errCh <- nil
	}()

	client, err := Dial(context.Background(), socketPath, "v1")
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	clone, err := client.Clone(context.Background())
	if err != nil {
		_ = client.Close()
		t.Fatalf("Clone: %v", err)
	}
	if err := clone.Close(); err != nil {
		t.Fatalf("clone.Close: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}
}
