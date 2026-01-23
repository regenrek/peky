package nl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestReadPromptFromArgs(t *testing.T) {
	ctx := root.CommandContext{
		Args: []string{"hello", "world"},
		Cmd:  &cli.Command{},
	}
	out, err := readPrompt(ctx)
	if err != nil {
		t.Fatalf("readPrompt error: %v", err)
	}
	if out != "hello world" {
		t.Fatalf("prompt=%q", out)
	}
}

func TestReadPromptFromStdin(t *testing.T) {
	cmd := &cli.Command{Flags: []cli.Flag{&cli.BoolFlag{Name: "stdin"}}}
	if err := cmd.Set("stdin", "true"); err != nil {
		t.Fatalf("cmd.Set(stdin) error: %v", err)
	}
	ctx := root.CommandContext{
		Cmd:   cmd,
		Stdin: strings.NewReader(" hello "),
	}
	out, err := readPrompt(ctx)
	if err != nil {
		t.Fatalf("readPrompt error: %v", err)
	}
	if out != "hello" {
		t.Fatalf("prompt=%q", out)
	}
}

func TestBuildPlanRulesAndSlash(t *testing.T) {
	plan, err := buildPlan("list sessions")
	if err != nil {
		t.Fatalf("buildPlan error: %v", err)
	}
	if len(plan.Commands) != 1 || plan.Commands[0].ID != "session.list" {
		t.Fatalf("plan=%#v", plan)
	}
	plan, err = buildPlan("/all hello team")
	if err != nil {
		t.Fatalf("buildPlan slash error: %v", err)
	}
	if len(plan.Commands) != 1 || plan.Commands[0].ID != "pane.send" {
		t.Fatalf("slash plan=%#v", plan)
	}
	if plan.Commands[0].Flags["scope"] != "all" || plan.Commands[0].Flags["text"] != "hello team" {
		t.Fatalf("slash flags=%#v", plan.Commands[0].Flags)
	}
}

func TestPlanFromTokens(t *testing.T) {
	specDoc, err := spec.LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault error: %v", err)
	}
	cmd, _, err := planFromTokens(specDoc, []string{"layouts"})
	if err != nil {
		t.Fatalf("planFromTokens error: %v", err)
	}
	if cmd.ID != "layouts.list" {
		t.Fatalf("cmd=%#v", cmd)
	}
}

func TestHasNLCommand(t *testing.T) {
	cmds := []output.NLPlannedCommand{
		{ID: "layouts.list"},
		{ID: "nl.plan"},
	}
	if !hasNLCommand(cmds) {
		t.Fatalf("expected nl command")
	}
}

func TestParseActionResultInvalid(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteString("nope")
	if got := parseActionResult(buf); got != nil {
		t.Fatalf("expected nil for invalid json")
	}
}
