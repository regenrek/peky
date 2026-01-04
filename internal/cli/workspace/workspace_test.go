package workspace

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/layout"
	ws "github.com/regenrek/peakypanes/internal/workspace"
)

func testCommand() *cli.Command {
	return &cli.Command{
		Name: "workspace",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id"},
			&cli.StringFlag{Name: "name"},
			&cli.StringFlag{Name: "path"},
		},
	}
}

func TestWorkspaceCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfgPath := mustDefaultConfigPath(t)
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{
			{Name: "ProjA", Path: "/tmp/proja"},
			{Name: "ProjB", Path: "/tmp/projb"},
		},
	}
	mustSaveConfig(t, cfgPath, cfg)

	deps := root.Dependencies{Version: "test"}
	mustContainAll(t, mustRunWorkspaceList(t, deps, false), "ProjA", "ProjB")
	mustContain(t, mustRunWorkspaceList(t, deps, true), "\"projects\"")

	mustRunWorkspaceClose(t, deps, "ProjA", false)
	mustRunWorkspaceClose(t, deps, "ProjA", true)
	mustHiddenProjectsAtLeast(t, mustLoadConfig(t, cfgPath), 1)

	mustRunWorkspaceOpen(t, deps, "ProjA")
	mustHiddenProjectsExactly(t, mustLoadConfig(t, cfgPath), 0)

	mustRunWorkspaceCloseAll(t, deps, false)
	mustRunWorkspaceCloseAll(t, deps, true)
	mustHiddenProjectsAtLeast(t, mustLoadConfig(t, cfgPath), 2)
}

func mustDefaultConfigPath(t *testing.T) string {
	t.Helper()
	cfgPath, err := layout.DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}
	return cfgPath
}

func mustSaveConfig(t *testing.T, cfgPath string, cfg *layout.Config) {
	t.Helper()
	if err := ws.SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}
}

func mustLoadConfig(t *testing.T, cfgPath string) *layout.Config {
	t.Helper()
	loaded, err := ws.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	return loaded
}

func mustRunWorkspaceList(t *testing.T, deps root.Dependencies, json bool) string {
	t.Helper()
	var out bytes.Buffer
	ctx := root.CommandContext{
		Context: context.Background(),
		Cmd:     testCommand(),
		Deps:    deps,
		Out:     &out,
		ErrOut:  &out,
		Stdin:   strings.NewReader(""),
		JSON:    json,
	}
	if err := runList(ctx); err != nil {
		t.Fatalf("runList(json=%v) error: %v", json, err)
	}
	return out.String()
}

func mustRunWorkspaceClose(t *testing.T, deps root.Dependencies, name string, json bool) {
	t.Helper()
	cmd := testCommand()
	_ = cmd.Set("name", name)
	ctx := root.CommandContext{
		Context: context.Background(),
		Cmd:     cmd,
		Deps:    deps,
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
		JSON:    json,
	}
	if err := runClose(ctx); err != nil {
		t.Fatalf("runClose(json=%v) error: %v", json, err)
	}
}

func mustRunWorkspaceOpen(t *testing.T, deps root.Dependencies, name string) {
	t.Helper()
	cmd := testCommand()
	_ = cmd.Set("name", name)
	ctx := root.CommandContext{
		Context: context.Background(),
		Cmd:     cmd,
		Deps:    deps,
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runOpen(ctx); err != nil {
		t.Fatalf("runOpen() error: %v", err)
	}
}

func mustRunWorkspaceCloseAll(t *testing.T, deps root.Dependencies, json bool) {
	t.Helper()
	ctx := root.CommandContext{
		Context: context.Background(),
		Cmd:     testCommand(),
		Deps:    deps,
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
		JSON:    json,
	}
	if err := runCloseAll(ctx); err != nil {
		t.Fatalf("runCloseAll(json=%v) error: %v", json, err)
	}
}

func mustContain(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("output missing %q, got %q", want, got)
	}
}

func mustContainAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		mustContain(t, got, want)
	}
}

func mustHiddenProjectsAtLeast(t *testing.T, cfg *layout.Config, wantMin int) {
	t.Helper()
	if len(cfg.Dashboard.HiddenProjects) < wantMin {
		t.Fatalf("expected >= %d hidden projects, got %#v", wantMin, cfg.Dashboard.HiddenProjects)
	}
}

func mustHiddenProjectsExactly(t *testing.T, cfg *layout.Config, want int) {
	t.Helper()
	if len(cfg.Dashboard.HiddenProjects) != want {
		t.Fatalf("expected %d hidden projects, got %#v", want, cfg.Dashboard.HiddenProjects)
	}
}
