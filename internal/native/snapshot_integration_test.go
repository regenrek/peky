//go:build integration

package native

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/terminal"
)

func TestSnapshotUsesPlainPreview(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	win, err := terminal.NewWindow(terminal.Options{
		ID:      "pane-1",
		Command: "sh",
		Args: []string{
			"-c",
			"i=0; while true; do i=$((i+1)); printf 'tick-%06d\n' $i; done",
		},
		Cols: 40,
		Rows: 6,
	})
	if err != nil {
		t.Fatalf("NewWindow error: %v", err)
	}
	t.Cleanup(func() {
		_ = win.Close()
	})

	for {
		if ctx.Err() != nil {
			t.Fatalf("timeout waiting for preview")
		}
		lines, _ := win.PreviewPlainLines(3)
		if len(lines) != 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	m := newTestManager(t)
	pane := &Pane{ID: "pane-1", Index: "0", window: win}
	session := &Session{Name: "sess", Panes: []*Pane{pane}}
	m.sessions["sess"] = session
	m.panes[pane.ID] = pane

	snaps := m.Snapshot(ctx, 2)
	if len(snaps) != 1 || len(snaps[0].Panes) != 1 {
		t.Fatalf("Snapshot() returned %d sessions", len(snaps))
	}
	preview := strings.Join(snaps[0].Panes[0].Preview, "\n")
	if strings.TrimSpace(preview) == "" {
		t.Fatalf("expected preview from plain lines, got empty")
	}
}
