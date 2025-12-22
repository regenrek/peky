package zellijctl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestListSessionsParsesOutput(t *testing.T) {
	client := &Client{
		bin: "zellij",
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return mockCmd(ctx, "alpha\nbeta\n", 0)
		},
	}

	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 2 || sessions[0] != "alpha" || sessions[1] != "beta" {
		t.Fatalf("unexpected sessions: %#v", sessions)
	}
}

func TestListSessionsHandlesNoSessions(t *testing.T) {
	client := &Client{
		bin: "zellij",
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return mockCmd(ctx, "No sessions", 1)
		},
	}

	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no sessions, got %#v", sessions)
	}
}

func TestListSessionsHandlesNoActiveSessions(t *testing.T) {
	client := &Client{
		bin: "zellij",
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return mockCmd(ctx, "No active zellij sessions found.", 1)
		},
	}

	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no sessions, got %#v", sessions)
	}
}

func TestSnapshotUsesPipe(t *testing.T) {
	json := `{"ok":true,"sessions":[{"name":"alpha","tabs":[],"panes":{"panes":{}},"connected_clients":1,"is_current_session":true}]}`
	client := &Client{
		bin: "zellij",
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			if containsArg(args, "list-sessions") {
				return mockCmd(ctx, "alpha\n", 0)
			}
			if containsArg(args, "pipe") {
				return mockCmd(ctx, json, 0)
			}
			return mockCmd(ctx, "", 1)
		},
	}

	sessions, err := client.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Name != "alpha" {
		t.Fatalf("unexpected snapshot: %#v", sessions)
	}
}

func TestPipeIncludesBridgePlugin(t *testing.T) {
	pluginPath := filepath.Join(t.TempDir(), "bridge.wasm")
	if err := os.WriteFile(pluginPath, []byte("wasm"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	var sawPlugin bool
	client := &Client{
		bin:        "zellij",
		bridgePath: pluginPath,
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			for i := 0; i < len(args); i++ {
				if args[i] == "--plugin" && i+1 < len(args) {
					if args[i+1] == normalizePluginURL(pluginPath) {
						sawPlugin = true
					}
				}
			}
			return mockCmd(ctx, `{"ok":true}`, 0)
		},
	}

	if err := client.SendKeys(context.Background(), "alpha", 1, "ls"); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}
	if !sawPlugin {
		t.Fatalf("expected pipe to include --plugin %q", normalizePluginURL(pluginPath))
	}
}

func TestClientActionsViaPipe(t *testing.T) {
	client := &Client{
		bin: "zellij",
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return mockCmd(ctx, `{"ok":true,"lines":["one","two"]}`, 0)
		},
	}

	ctx := context.Background()
	lines, err := client.CapturePaneLines(ctx, "alpha", 1, 2)
	if err != nil {
		t.Fatalf("CapturePaneLines: %v", err)
	}
	if len(lines) != 2 || lines[0] != "one" || lines[1] != "two" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
	if err := client.SendKeys(ctx, "alpha", 1, "ls"); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}
	if err := client.RenameSession(ctx, "alpha", "beta"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}
	if err := client.RenameTab(ctx, "alpha", 1, "tab"); err != nil {
		t.Fatalf("RenameTab: %v", err)
	}
	if err := client.SwitchSession(ctx, "alpha", "beta", nil); err != nil {
		t.Fatalf("SwitchSession: %v", err)
	}
}

func TestKillAndAttachSession(t *testing.T) {
	client := &Client{
		bin: "zellij",
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return mockCmd(ctx, "", 0)
		},
	}

	ctx := context.Background()
	if err := client.KillSession(ctx, "alpha"); err != nil {
		t.Fatalf("KillSession: %v", err)
	}
	if err := client.AttachSession(ctx, "alpha"); err != nil {
		t.Fatalf("AttachSession: %v", err)
	}
}

func TestParseTabPosition(t *testing.T) {
	if _, err := ParseTabPosition(""); err == nil {
		t.Fatalf("expected error for empty tab position")
	}
	if _, err := ParseTabPosition("nope"); err == nil {
		t.Fatalf("expected error for invalid tab position")
	}
	if got, err := ParseTabPosition("3"); err != nil || got != 3 {
		t.Fatalf("expected 3, got %d (err=%v)", got, err)
	}
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func mockCmd(ctx context.Context, out string, exitCode int) *exec.Cmd {
	script := "printf '%s' \"$MOCK_OUT\"; exit $MOCK_EXIT"
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MOCK_OUT=%s", out),
		fmt.Sprintf("MOCK_EXIT=%d", exitCode),
	)
	return cmd
}

func TestSnapshotCache(t *testing.T) {
	calls := 0
	json := `{"ok":true,"sessions":[{"name":"alpha","tabs":[],"panes":{"panes":{}},"connected_clients":1,"is_current_session":true}]}`
	client := &Client{
		bin: "zellij",
		run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			calls++
			if containsArg(args, "list-sessions") {
				return mockCmd(ctx, "alpha\n", 0)
			}
			return mockCmd(ctx, json, 0)
		},
	}

	ctx := context.Background()
	if _, err := client.Snapshot(ctx); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls after first snapshot, got %d", calls)
	}
	if _, err := client.Snapshot(ctx); err != nil {
		t.Fatalf("Snapshot cache: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected cache hit to avoid extra calls, got %d", calls)
	}

	// Force cache expiry
	client.cacheAt = time.Now().Add(-time.Second)
	if _, err := client.Snapshot(ctx); err != nil {
		t.Fatalf("Snapshot after cache expiry: %v", err)
	}
	if calls <= 2 {
		t.Fatalf("expected cache miss to call run, got %d", calls)
	}
}
