package app

import (
	"context"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/cli/spec"
	"github.com/regenrek/peakypanes/internal/tui/icons"
)

func TestPekyPolicyAllowsPatterns(t *testing.T) {
	p := pekyPolicy{allowed: []string{"auth.*", "model"}}
	if !p.allows("auth.login") {
		t.Fatalf("expected auth.login allowed")
	}
	if p.allows("") {
		t.Fatalf("expected empty command blocked")
	}

	p = pekyPolicy{blocked: []string{"pane.*"}}
	if p.allows("pane.send") {
		t.Fatalf("expected pane.send blocked")
	}
	if !p.allows("help") {
		t.Fatalf("expected help allowed when not blocked")
	}
}

func TestPekyContextAndWorkDir(t *testing.T) {
	m := newTestModelLite()
	pane := m.selectedPane()
	if pane == nil {
		t.Fatalf("expected selected pane")
		return
	}
	pane.Cwd = "/alpha/pane"
	ctx := m.pekyContext()
	if !strings.Contains(ctx, "Selected project") || !strings.Contains(ctx, "Selected session") {
		t.Fatalf("expected context lines, got %q", ctx)
	}
	workDir, err := m.pekyWorkDir()
	if err != nil {
		t.Fatalf("pekyWorkDir error: %v", err)
	}
	if workDir != "/alpha/pane" {
		t.Fatalf("expected pane cwd, got %q", workDir)
	}
	pane.Cwd = "unknown"
	workDir, err = m.pekyWorkDir()
	if err != nil {
		t.Fatalf("pekyWorkDir fallback error: %v", err)
	}
	if workDir == "" {
		t.Fatalf("expected workDir fallback")
	}
}

func TestPekySpinnerAndLabel(t *testing.T) {
	t.Setenv("PEKY_ICON_SET", "ascii")
	frames := icons.Active().Spinner
	if len(frames) == 0 {
		t.Fatalf("expected spinner frames")
	}
	m := newTestModelLite()
	if frame := m.pekySpinnerFrame(); frame == "" {
		t.Fatalf("expected spinner frame")
	}
	if cmd := m.handlePekySpinnerTick(); cmd != nil {
		t.Fatalf("expected nil cmd when not busy")
	}
	m.pekyBusy = true
	prev := m.pekySpinnerIndex
	cmd := m.handlePekySpinnerTick()
	if cmd == nil {
		t.Fatalf("expected spinner tick cmd")
	}
	if m.pekySpinnerIndex == prev {
		t.Fatalf("expected spinner index advance")
	}
}

func TestPekySetupHintAndToast(t *testing.T) {
	if pekySetupHint(agent.ProviderOpenAI) == "" {
		t.Fatalf("expected openai hint")
	}
	if pekySetupHint("unknown") == "" {
		t.Fatalf("expected default hint")
	}
	if pekySuccessToast("") != "Done" {
		t.Fatalf("expected Done toast")
	}
	long := strings.Repeat("a", 200)
	toast := pekySuccessToast(long)
	if len(toast) > 120 {
		t.Fatalf("expected toast trimmed")
	}
}

func TestPekyCommandHelpers(t *testing.T) {
	cmds := []spec.Command{
		{ID: "dashboard", Name: "dashboard"},
		{ID: "alpha", Name: "alpha", Subcommands: []spec.Command{{ID: "alpha.beta", Name: "beta"}}},
		{ID: "gamma", Name: "gamma", Aliases: []string{"g"}},
	}
	filtered := filterPekyCommands(cmds)
	if len(filtered) != 2 {
		t.Fatalf("expected filtered commands, got %d", len(filtered))
	}
	cmd, consumed := matchCommandSpec(filtered, []string{"alpha", "beta"})
	if cmd == nil || cmd.ID != "alpha.beta" || consumed != 2 {
		t.Fatalf("unexpected match result: %#v consumed=%d", cmd, consumed)
	}
	if !matchesCommandToken(spec.Command{Name: "gamma", Aliases: []string{"g"}}, "g") {
		t.Fatalf("expected alias match")
	}
	if !hasYesFlag([]string{"help", "--yes"}) {
		t.Fatalf("expected yes flag")
	}
}

func TestRunPekyCommandHelp(t *testing.T) {
	output, err := runPekyCommand(context.Background(), pekyPolicy{}, pekyToolInput{Command: "help"}, "", "test")
	if err != nil {
		t.Fatalf("runPekyCommand error: %v", err)
	}
	if strings.TrimSpace(output) == "" {
		t.Fatalf("expected output")
	}
}

func TestHandlePekyResultBranches(t *testing.T) {
	m := newTestModelLite()
	m.pekyRunID = 1
	m.pekyBusy = true

	m.handlePekyResult(pekyResultMsg{RunID: 1, Err: agent.ErrAuthMissing})
	if m.toast.Text == "" {
		t.Fatalf("expected toast on auth missing")
	}

	m.pekyRunID = 2
	m.handlePekyResult(pekyResultMsg{RunID: 2, Err: context.Canceled})
	if m.toast.Text == "" {
		t.Fatalf("expected toast on cancel")
	}

	m.pekyRunID = 3
	m.pekyBusy = true
	m.handlePekyResult(pekyResultMsg{RunID: 3, Text: "ok", History: []agent.Message{{Role: "user"}}})
	if m.pekyPromptLine == "" {
		t.Fatalf("expected prompt line")
	}
	if len(m.pekyMessages) == 0 {
		t.Fatalf("expected history set")
	}
}
