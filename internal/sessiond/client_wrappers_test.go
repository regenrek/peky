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
	defer cancel()
	client, err := Dial(ctx, socket, "test")
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if got := client.Version(); got != "test" {
		t.Fatalf("Version=%q want=%q", got, "test")
	}

	if err := client.RenamePaneByID(ctx, "missing", "title"); err == nil {
		t.Fatalf("expected RenamePaneByID error")
	}
	if err := client.ClosePaneByID(ctx, "missing"); err == nil {
		t.Fatalf("expected ClosePaneByID error")
	}
	if err := client.SendInputAction(ctx, "missing", []byte("x"), "action", "summary"); err == nil {
		t.Fatalf("expected SendInputAction error")
	}
	if _, err := client.SendInputScope(ctx, "", []byte("x")); err == nil {
		t.Fatalf("expected SendInputScope error")
	}
	if _, err := client.SendInputScopeAction(ctx, "", []byte("x"), "action", "summary"); err == nil {
		t.Fatalf("expected SendInputScopeAction error")
	}

	if _, err := client.PaneOutput(ctx, PaneOutputRequest{}); err == nil {
		t.Fatalf("expected PaneOutput error")
	}
	if _, err := client.PaneSnapshot(ctx, "", 10); err == nil {
		t.Fatalf("expected PaneSnapshot error")
	}
	if _, err := client.PaneHistory(ctx, PaneHistoryRequest{}); err == nil {
		t.Fatalf("expected PaneHistory error")
	}
	if _, err := client.PaneWait(ctx, PaneWaitRequest{}); err == nil {
		t.Fatalf("expected PaneWait error")
	}

	if _, err := client.PaneTags(ctx, ""); err == nil {
		t.Fatalf("expected PaneTags error")
	}
	if _, err := client.AddPaneTags(ctx, "", []string{"tag"}); err == nil {
		t.Fatalf("expected AddPaneTags error")
	}
	if _, err := client.RemovePaneTags(ctx, "", []string{"tag"}); err == nil {
		t.Fatalf("expected RemovePaneTags error")
	}

	if err := client.FocusSession(ctx, ""); err == nil {
		t.Fatalf("expected FocusSession error")
	}
	if err := client.FocusPane(ctx, ""); err == nil {
		t.Fatalf("expected FocusPane error")
	}
	if err := client.SignalPane(ctx, "", "TERM"); err == nil {
		t.Fatalf("expected SignalPane error")
	}

	if _, err := client.RelayCreate(ctx, RelayConfig{}); err == nil {
		t.Fatalf("expected RelayCreate error")
	}
	if relays, err := client.RelayList(ctx); err != nil || len(relays) != 0 {
		t.Fatalf("RelayList relays=%v err=%v", relays, err)
	}
	if err := client.RelayStop(ctx, ""); err == nil {
		t.Fatalf("expected RelayStop error")
	}
	if err := client.RelayStopAll(ctx); err != nil {
		t.Fatalf("RelayStopAll: %v", err)
	}

	if resp, err := client.EventsReplay(ctx, EventsReplayRequest{Limit: 10}); err != nil {
		t.Fatalf("EventsReplay: %v", err)
	} else if len(resp.Events) != 0 {
		t.Fatalf("EventsReplay events=%v want empty", resp.Events)
	}
}
