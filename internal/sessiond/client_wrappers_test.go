package sessiond

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestClientWrapperMethodsHitDaemonHandlers(t *testing.T) {
	tc := newDaemonTestClient(t)
	assertClientWrapperBasics(t, tc)
	assertClientWrapperMissingTargetErrors(t, tc)
	assertClientWrapperRelayMethods(t, tc)
	assertClientWrapperEventsReplay(t, tc)
}

type daemonTestClient struct {
	socket string
	client *Client
	ctx    context.Context
}

func newDaemonTestClient(t *testing.T) daemonTestClient {
	t.Helper()

	base := t.TempDir()
	if runtime.GOOS != "windows" {
		if dir, err := os.MkdirTemp("/tmp", "ppd-"); err == nil {
			base = dir
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
		}
	}
	socket := filepath.Join(base, "daemon.sock")
	pid := filepath.Join(base, "daemon.pid")

	daemon, err := NewDaemon(DaemonConfig{
		Version:       "test",
		SocketPath:    socket,
		PidPath:       pid,
		HandleSignals: false,
	})
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(func() { _ = daemon.Stop() })
	if err := daemon.Start(); err != nil {
		t.Fatalf("daemon.Start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	client, err := Dial(ctx, socket, "test")
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return daemonTestClient{socket: socket, client: client, ctx: ctx}
}

func assertClientWrapperBasics(t *testing.T, tc daemonTestClient) {
	t.Helper()
	if got := tc.client.Version(); got != "test" {
		t.Fatalf("Version=%q want=%q", got, "test")
	}
}

func assertClientWrapperMissingTargetErrors(t *testing.T, tc daemonTestClient) {
	t.Helper()

	calls := []struct {
		name string
		fn   func() error
	}{
		{name: "RenamePaneByID", fn: func() error { return tc.client.RenamePaneByID(tc.ctx, "missing", "title") }},
		{name: "ClosePaneByID", fn: func() error { return tc.client.ClosePaneByID(tc.ctx, "missing") }},
		{name: "SendInputAction", fn: func() error { return tc.client.SendInputAction(tc.ctx, "missing", []byte("x"), "action", "summary") }},
		{name: "SendInputScope", fn: func() error { _, err := tc.client.SendInputScope(tc.ctx, "", []byte("x")); return err }},
		{name: "SendInputScopeAction", fn: func() error {
			_, err := tc.client.SendInputScopeAction(tc.ctx, "", []byte("x"), "action", "summary")
			return err
		}},
		{name: "PaneOutput", fn: func() error { _, err := tc.client.PaneOutput(tc.ctx, PaneOutputRequest{}); return err }},
		{name: "PaneSnapshot", fn: func() error { _, err := tc.client.PaneSnapshot(tc.ctx, "", 10); return err }},
		{name: "PaneHistory", fn: func() error { _, err := tc.client.PaneHistory(tc.ctx, PaneHistoryRequest{}); return err }},
		{name: "PaneWait", fn: func() error { _, err := tc.client.PaneWait(tc.ctx, PaneWaitRequest{}); return err }},
		{name: "PaneTags", fn: func() error { _, err := tc.client.PaneTags(tc.ctx, ""); return err }},
		{name: "AddPaneTags", fn: func() error { _, err := tc.client.AddPaneTags(tc.ctx, "", []string{"tag"}); return err }},
		{name: "RemovePaneTags", fn: func() error { _, err := tc.client.RemovePaneTags(tc.ctx, "", []string{"tag"}); return err }},
		{name: "FocusSession", fn: func() error { return tc.client.FocusSession(tc.ctx, "") }},
		{name: "FocusPane", fn: func() error { return tc.client.FocusPane(tc.ctx, "") }},
		{name: "SignalPane", fn: func() error { return tc.client.SignalPane(tc.ctx, "", "TERM") }},
		{name: "RelayCreate", fn: func() error { _, err := tc.client.RelayCreate(tc.ctx, RelayConfig{}); return err }},
		{name: "RelayStop", fn: func() error { return tc.client.RelayStop(tc.ctx, "") }},
	}
	for _, call := range calls {
		if err := call.fn(); err == nil {
			t.Fatalf("expected %s error", call.name)
		}
	}
}

func assertClientWrapperRelayMethods(t *testing.T, tc daemonTestClient) {
	t.Helper()

	relays, err := tc.client.RelayList(tc.ctx)
	if err != nil || len(relays) != 0 {
		t.Fatalf("RelayList relays=%v err=%v", relays, err)
	}
	if err := tc.client.RelayStopAll(tc.ctx); err != nil {
		t.Fatalf("RelayStopAll: %v", err)
	}
}

func assertClientWrapperEventsReplay(t *testing.T, tc daemonTestClient) {
	t.Helper()

	resp, err := tc.client.EventsReplay(tc.ctx, EventsReplayRequest{Limit: 10})
	if err != nil {
		t.Fatalf("EventsReplay: %v", err)
	}
	if len(resp.Events) != 0 {
		t.Fatalf("EventsReplay events=%v want empty", resp.Events)
	}
}
