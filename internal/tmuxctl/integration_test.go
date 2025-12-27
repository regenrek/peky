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

	t.Setenv("TMUX_TMPDIR", shortTmpDir(t))

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

func TestIntegrationSendKeysAndCapture(t *testing.T) {
	if os.Getenv("PEAKYPANES_INTEGRATION") != "1" {
		t.Skip("integration test disabled; set PEAKYPANES_INTEGRATION=1")
	}

	t.Setenv("TMUX_TMPDIR", shortTmpDir(t))

	client, err := NewClient("")
	if err != nil {
		t.Skipf("tmux not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session := fmt.Sprintf("peakypanes-smoke-%d", time.Now().UnixNano())
	startDir := t.TempDir()
	paneID, err := client.NewSessionWithCmd(ctx, session, startDir, "smoke", "sh -c 'printf \"ready\\n\"; sleep 5'")
	if err != nil {
		t.Fatalf("NewSessionWithCmd() error: %v", err)
	}
	if strings.TrimSpace(paneID) == "" {
		t.Fatalf("NewSessionWithCmd() returned empty pane id")
	}
	t.Cleanup(func() {
		_ = client.KillSession(context.Background(), session)
	})

	waitForPaneLine(t, client, ctx, paneID, "ready")

	if err := client.SendKeysLiteral(ctx, paneID, "echo smoke"); err != nil {
		t.Fatalf("SendKeysLiteral() error: %v", err)
	}
	if err := client.SendKeys(ctx, paneID, "Enter"); err != nil {
		t.Fatalf("SendKeys(Enter) error: %v", err)
	}

	waitForPaneLine(t, client, ctx, paneID, "smoke")
}
func shortTmpDir(t *testing.T) string {
	t.Helper()
	base := "/tmp"
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		base = os.TempDir()
	}
	dir, err := os.MkdirTemp(base, "peakypanes-tmux-")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

func containsSession(sessions []string, name string) bool {
	for _, session := range sessions {
		if strings.TrimSpace(session) == name {
			return true
		}
	}
	return false
}

func waitForPaneLine(t *testing.T, client *Client, ctx context.Context, target, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		lines, err := client.CapturePaneLines(ctx, target, 50)
		if err == nil {
			for _, line := range lines {
				if strings.Contains(line, want) {
					return
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in pane %s", want, target)
}
