package tmuxctl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestIntegrationEnsureSessionLifecycle(t *testing.T) {
	if os.Getenv("PEAKYPANES_INTEGRATION") != "1" {
		t.Skip("integration test disabled; set PEAKYPANES_INTEGRATION=1")
	}

	t.Setenv("TMUX_TMPDIR", t.TempDir())

	client, err := NewClient("")
	if err != nil {
		t.Skipf("tmux not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := fmt.Sprintf("peakypanes-test-%d", time.Now().UnixNano())
	startDir := t.TempDir()
	t.Cleanup(func() {
		_ = client.KillSession(context.Background(), session)
	})

	res, err := client.EnsureSession(ctx, Options{
		Session:  session,
		Layout:   layout.Grid{Rows: 1, Columns: 2},
		StartDir: startDir,
		Attach:   false,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("EnsureSession(create) error: %v", err)
	}
	if !res.Created || res.Attached {
		t.Fatalf("EnsureSession(create) result = %#v", res)
	}

	sessions, err := client.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if !containsSession(sessions, session) {
		t.Fatalf("session %q not found in %#v", session, sessions)
	}

	res, err = client.EnsureSession(ctx, Options{
		Session:  session,
		Layout:   layout.Grid{Rows: 1, Columns: 2},
		StartDir: startDir,
		Attach:   false,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("EnsureSession(existing) error: %v", err)
	}
	if res.Created || res.Attached {
		t.Fatalf("EnsureSession(existing) result = %#v", res)
	}

	if err := client.KillSession(ctx, session); err != nil {
		t.Fatalf("KillSession() error: %v", err)
	}
	sessions, err = client.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions(after kill) error: %v", err)
	}
	if containsSession(sessions, session) {
		t.Fatalf("session %q still present after kill", session)
	}
}

func containsSession(sessions []string, name string) bool {
	for _, session := range sessions {
		if strings.TrimSpace(session) == name {
			return true
		}
	}
	return false
}
