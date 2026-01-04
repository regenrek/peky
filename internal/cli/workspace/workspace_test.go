package workspace

import (
	"bytes"
	"context"
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
	cfgPath, err := layout.DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error: %v", err)
	}
	cfg := &layout.Config{
		Projects: []layout.ProjectConfig{
			{Name: "ProjA", Path: "/tmp/proja"},
			{Name: "ProjB", Path: "/tmp/projb"},
		},
	}
	if err := ws.SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	var listOut bytes.Buffer
	listCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     testCommand(),
		Deps:    root.Dependencies{Version: "test"},
		Out:     &listOut,
		ErrOut:  &listOut,
		Stdin:   strings.NewReader(""),
	}
	if err := runList(listCtx); err != nil {
		t.Fatalf("runList() error: %v", err)
	}
	if !strings.Contains(listOut.String(), "ProjA") || !strings.Contains(listOut.String(), "ProjB") {
		t.Fatalf("runList output = %q", listOut.String())
	}
	var listJSON bytes.Buffer
	listJSONCtx := listCtx
	listJSONCtx.JSON = true
	listJSONCtx.Out = &listJSON
	if err := runList(listJSONCtx); err != nil {
		t.Fatalf("runList(json) error: %v", err)
	}
	if !strings.Contains(listJSON.String(), "\"projects\"") {
		t.Fatalf("runList(json) output = %q", listJSON.String())
	}

	closeCmd := testCommand()
	_ = closeCmd.Set("name", "ProjA")
	closeCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     closeCmd,
		Deps:    root.Dependencies{Version: "test"},
		Out:     &listOut,
		ErrOut:  &listOut,
		Stdin:   strings.NewReader(""),
	}
	if err := runClose(closeCtx); err != nil {
		t.Fatalf("runClose() error: %v", err)
	}
	closeJSON := closeCtx
	closeJSON.JSON = true
	closeJSON.Out = &listOut
	if err := runClose(closeJSON); err != nil {
		t.Fatalf("runClose(json) error: %v", err)
	}
	loaded, err := ws.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if len(loaded.Dashboard.HiddenProjects) == 0 {
		t.Fatalf("expected hidden project after close")
	}

	openCmd := testCommand()
	_ = openCmd.Set("name", "ProjA")
	openCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     openCmd,
		Deps:    root.Dependencies{Version: "test"},
		Out:     &listOut,
		ErrOut:  &listOut,
		Stdin:   strings.NewReader(""),
	}
	if err := runOpen(openCtx); err != nil {
		t.Fatalf("runOpen() error: %v", err)
	}
	loaded, err = ws.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if len(loaded.Dashboard.HiddenProjects) != 0 {
		t.Fatalf("expected hidden projects cleared, got %#v", loaded.Dashboard.HiddenProjects)
	}

	closeAllCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     testCommand(),
		Deps:    root.Dependencies{Version: "test"},
		Out:     &listOut,
		ErrOut:  &listOut,
		Stdin:   strings.NewReader(""),
	}
	if err := runCloseAll(closeAllCtx); err != nil {
		t.Fatalf("runCloseAll() error: %v", err)
	}
	closeAllJSON := closeAllCtx
	closeAllJSON.JSON = true
	closeAllJSON.Out = &listOut
	if err := runCloseAll(closeAllJSON); err != nil {
		t.Fatalf("runCloseAll(json) error: %v", err)
	}
	loaded, err = ws.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if len(loaded.Dashboard.HiddenProjects) < 2 {
		t.Fatalf("expected hidden projects after close-all, got %#v", loaded.Dashboard.HiddenProjects)
	}
}
