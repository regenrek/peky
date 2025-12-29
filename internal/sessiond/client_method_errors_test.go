package sessiond

import (
	"context"
	"testing"
)

func TestClientMethodErrorsWithoutConnection(t *testing.T) {
	client := &Client{pending: make(map[uint64]chan Envelope)}

	if _, err := client.SessionNames(context.Background()); err == nil {
		t.Fatalf("expected SessionNames error without connection")
	}
	if _, _, err := client.Snapshot(context.Background(), 1, 0); err == nil {
		t.Fatalf("expected Snapshot error without connection")
	}
	if _, err := client.StartSession(context.Background(), StartSessionRequest{Name: "demo", Path: "/tmp"}); err == nil {
		t.Fatalf("expected StartSession error without connection")
	}
	if _, err := client.RenameSession(context.Background(), "old", "new"); err == nil {
		t.Fatalf("expected RenameSession error without connection")
	}
	if _, err := client.SplitPane(context.Background(), "session", "1", true, 10); err == nil {
		t.Fatalf("expected SplitPane error without connection")
	}
	if _, err := client.GetPaneView(context.Background(), PaneViewRequest{PaneID: "pane", Cols: 1, Rows: 1}); err == nil {
		t.Fatalf("expected GetPaneView error without connection")
	}
	if _, err := client.TerminalAction(context.Background(), TerminalActionRequest{PaneID: "pane", Action: TerminalPageDown}); err == nil {
		t.Fatalf("expected TerminalAction error without connection")
	}
	if _, err := client.HandleTerminalKey(context.Background(), TerminalKeyRequest{PaneID: "pane", Key: "esc"}); err == nil {
		t.Fatalf("expected HandleTerminalKey error without connection")
	}
}
