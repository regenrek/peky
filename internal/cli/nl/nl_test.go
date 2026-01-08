package nl

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/spec"
	"github.com/regenrek/peakypanes/internal/identity"
)

func TestShellTokens(t *testing.T) {
	tokens, ok := shellTokens(identity.CLIName + " layouts")
	if !ok || len(tokens) != 1 || tokens[0] != "layouts" {
		t.Fatalf("tokens=%v ok=%v", tokens, ok)
	}
	if _, ok := shellTokens("-x"); ok {
		t.Fatalf("expected false for flag-leading prompt")
	}
}

func TestMatchCommand(t *testing.T) {
	specDoc, err := spec.LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault error: %v", err)
	}
	cmd, consumed := matchCommand(specDoc.Commands, []string{"layouts"})
	if cmd == nil || cmd.ID != "layouts.list" || consumed != 1 {
		t.Fatalf("cmd=%#v consumed=%d", cmd, consumed)
	}
}

func TestParseFlags(t *testing.T) {
	cmd := spec.Command{
		Name: "context",
		Flags: []spec.Flag{
			{Name: "include", Aliases: []string{"i"}, Type: "string", Repeatable: true},
			{Name: "json", Aliases: []string{"j"}, Type: "bool"},
		},
	}
	flags, args := parseFlags(cmd, []string{"--include", "git", "-i", "errors", "--json", "rest"})
	if len(args) != 1 || args[0] != "rest" {
		t.Fatalf("args=%v", args)
	}
	if v, ok := flags["json"].(bool); !ok || !v {
		t.Fatalf("flags=%#v", flags)
	}
	inc, ok := flags["include"].([]string)
	if !ok || len(inc) != 2 || inc[0] != "git" || inc[1] != "errors" {
		t.Fatalf("include=%#v", flags["include"])
	}
}

func TestBuildArgsAndInsertGlobalFlags(t *testing.T) {
	cmd := output.NLPlannedCommand{
		ID:      "pane.send",
		Command: "pane send",
		Flags: map[string]any{
			"scope": "all",
			"yes":   true,
			"to":    []string{"p1", "p2"},
		},
		Args: []string{"hello"},
	}
	args := buildArgs(cmd)
	if len(args) == 0 || args[0] != identity.CLIName {
		t.Fatalf("args=%v", args)
	}
	withGlobal := insertGlobalFlags(args, "--json")
	if len(withGlobal) < 2 || withGlobal[1] != "--json" {
		t.Fatalf("withGlobal=%v", withGlobal)
	}
}

func TestSupportsJSON(t *testing.T) {
	if !supportsJSON("layouts.list") {
		t.Fatalf("expected layouts.list supports JSON")
	}
	if supportsJSON("version") {
		t.Fatalf("expected version does not support JSON")
	}
}

func TestParseActionResult(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteString("{\"ok\":true,\"data\":{\"action\":\"x\",\"status\":\"ok\"}}")
	if got := parseActionResult(buf); got == nil || got.Action != "x" {
		t.Fatalf("got=%#v", got)
	}
}

func TestExecutePlanRunsLayouts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	prevExiter := cli.OsExiter
	cli.OsExiter = func(int) {}
	t.Cleanup(func() { cli.OsExiter = prevExiter })

	ctx := root.CommandContext{
		Context: context.Background(),
		Deps:    root.Dependencies{Version: "test"},
		Stdin:   strings.NewReader(""),
		ErrOut:  io.Discard,
	}
	plan := output.NLPlan{
		PlanID: "plan-test",
		Commands: []output.NLPlannedCommand{{
			ID:      "layouts.list",
			Command: "layouts",
		}},
	}
	exec, err := executePlan(ctx, plan)
	if err != nil {
		t.Fatalf("executePlan error: %v", err)
	}
	if len(exec.Steps) != 1 || exec.Steps[0].Status != "ok" {
		t.Fatalf("exec=%#v", exec)
	}
}
