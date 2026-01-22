package debug

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/runenv"
)

func TestRunPathsText(t *testing.T) {
	base := t.TempDir()
	runtimeDir := filepath.Join(base, "runtime")
	dataDir := filepath.Join(base, "data")
	configDir := filepath.Join(base, "config")

	t.Setenv(runenv.RuntimeDirEnv, runtimeDir)
	t.Setenv(runenv.DataDirEnv, dataDir)
	t.Setenv(runenv.ConfigDirEnv, configDir)
	t.Setenv(runenv.FreshConfigEnv, "")

	var out bytes.Buffer
	ctx := root.CommandContext{
		Deps: root.Dependencies{Version: "test"},
		Out:  &out,
		Cmd:  &cli.Command{Name: "debug"},
	}
	if err := runPaths(ctx); err != nil {
		t.Fatalf("runPaths error: %v", err)
	}
	got := out.String()
	wantConfigPath := filepath.Join(configDir, "config.yml")
	wantLayoutsDir := filepath.Join(configDir, "layouts")
	wantSocket := filepath.Join(runtimeDir, "daemon.sock")
	wantPid := filepath.Join(runtimeDir, "daemon.pid")
	wantNotice := filepath.Join(configDir, identity.RestartNoticeFlagFile)

	assertContains(t, got, "fresh_config: false\n")
	assertContains(t, got, "runtime_dir: "+runtimeDir+"\n")
	assertContains(t, got, "data_dir: "+dataDir+"\n")
	assertContains(t, got, "config_dir: "+configDir+"\n")
	assertContains(t, got, "config_path: "+wantConfigPath+"\n")
	assertContains(t, got, "layouts_dir: "+wantLayoutsDir+"\n")
	assertContains(t, got, "daemon_socket_path: "+wantSocket+"\n")
	assertContains(t, got, "daemon_pid_path: "+wantPid+"\n")
	assertContains(t, got, "restart_notice_path: "+wantNotice+"\n")
}

func TestRunPathsJSON(t *testing.T) {
	base := t.TempDir()
	runtimeDir := filepath.Join(base, "runtime")
	dataDir := filepath.Join(base, "data")
	configDir := filepath.Join(base, "config")

	t.Setenv(runenv.RuntimeDirEnv, runtimeDir)
	t.Setenv(runenv.DataDirEnv, dataDir)
	t.Setenv(runenv.ConfigDirEnv, configDir)
	t.Setenv(runenv.FreshConfigEnv, "")

	var out bytes.Buffer
	ctx := root.CommandContext{
		Deps: root.Dependencies{Version: "test"},
		Out:  &out,
		JSON: true,
		Cmd:  &cli.Command{Name: "debug"},
	}
	if err := runPaths(ctx); err != nil {
		t.Fatalf("runPaths error: %v", err)
	}
	var resp struct {
		Ok   bool              `json:"ok"`
		Data output.DebugPaths `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("expected ok response")
	}
	if resp.Meta.Command != "debug.paths" {
		t.Fatalf("meta.command=%q", resp.Meta.Command)
	}
	wantConfigPath := filepath.Join(configDir, "config.yml")
	wantLayoutsDir := filepath.Join(configDir, "layouts")
	wantSocket := filepath.Join(runtimeDir, "daemon.sock")
	wantPid := filepath.Join(runtimeDir, "daemon.pid")
	wantNotice := filepath.Join(configDir, identity.RestartNoticeFlagFile)

	if resp.Data.RuntimeDir != runtimeDir {
		t.Fatalf("runtime_dir=%q", resp.Data.RuntimeDir)
	}
	if resp.Data.DataDir != dataDir {
		t.Fatalf("data_dir=%q", resp.Data.DataDir)
	}
	if resp.Data.ConfigDir != configDir {
		t.Fatalf("config_dir=%q", resp.Data.ConfigDir)
	}
	if resp.Data.ConfigPath != wantConfigPath {
		t.Fatalf("config_path=%q", resp.Data.ConfigPath)
	}
	if resp.Data.LayoutsDir != wantLayoutsDir {
		t.Fatalf("layouts_dir=%q", resp.Data.LayoutsDir)
	}
	if resp.Data.DaemonSocketPath != wantSocket {
		t.Fatalf("daemon_socket_path=%q", resp.Data.DaemonSocketPath)
	}
	if resp.Data.DaemonPidPath != wantPid {
		t.Fatalf("daemon_pid_path=%q", resp.Data.DaemonPidPath)
	}
	if resp.Data.RestartNoticePath != wantNotice {
		t.Fatalf("restart_notice_path=%q", resp.Data.RestartNoticePath)
	}
	if resp.Data.FreshConfig {
		t.Fatalf("fresh_config=true")
	}
}

func assertContains(t *testing.T, value, substr string) {
	t.Helper()
	if !strings.Contains(value, substr) {
		t.Fatalf("expected %q in %q", substr, value)
	}
}
