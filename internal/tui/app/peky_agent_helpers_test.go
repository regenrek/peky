package app

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/agent"
	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestPekySetupHint(t *testing.T) {
	cases := []struct {
		provider agent.Provider
		want     string
	}{
		{provider: agent.ProviderOpenAI, want: "Set OPENAI_API_KEY."},
		{provider: agent.ProviderAnthropic, want: "Set ANTHROPIC_API_KEY."},
		{provider: agent.ProviderGitHubCopilot, want: "Authentication required for GitHub Copilot."},
		{provider: agent.Provider("unknown"), want: "Configure provider credentials in your environment."},
	}
	for _, tc := range cases {
		if got := pekySetupHint(tc.provider); got != tc.want {
			t.Fatalf("pekySetupHint(%q) = %q, want %q", tc.provider, got, tc.want)
		}
	}
}

func TestFormatPekyOutput(t *testing.T) {
	if got := formatPekyOutput("", "", nil); got != "ok" {
		t.Fatalf("empty output = %q", got)
	}
	if got := formatPekyOutput("", "", errors.New("boom")); got != "error: boom" {
		t.Fatalf("error-only output = %q", got)
	}
	got := formatPekyOutput("hello", "warn", errors.New("boom"))
	if !strings.Contains(got, "hello") || !strings.Contains(got, "stderr:\nwarn") || !strings.Contains(got, "error: boom") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestCliDeps_Defaults(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := cliDeps(stdout, stderr, "  /work  ", "")
	if deps.Version != "dev" {
		t.Fatalf("version = %q", deps.Version)
	}
	if deps.WorkDir != "/work" {
		t.Fatalf("workdir = %q", deps.WorkDir)
	}
	if deps.Stdout != stdout || deps.Stderr != stderr {
		t.Fatalf("stdout/stderr not wired")
	}
}

func TestPekyWorkDir(t *testing.T) {
	var nilModel *Model
	if _, err := nilModel.pekyWorkDir(); err == nil {
		t.Fatalf("expected error for nil model")
	}

	m := newTestModelLite()
	pane := m.selectedPane()
	pane.Cwd = "  /pane  "
	if got, err := m.pekyWorkDir(); err != nil || got != "/pane" {
		t.Fatalf("pane cwd = %q err=%v", got, err)
	}

	pane.Cwd = "unknown"
	session := m.selectedSession()
	session.Path = "  /session  "
	if got, err := m.pekyWorkDir(); err != nil || got != "/session" {
		t.Fatalf("session path = %q err=%v", got, err)
	}

	pane.Cwd = ""
	session.Path = ""
	if _, err := m.pekyWorkDir(); err == nil {
		t.Fatalf("expected error for missing paths")
	}
}

func TestFilterPekySpec(t *testing.T) {
	specDoc := &spec.Spec{
		App: spec.AppSpec{DefaultCommand: "dashboard"},
		Commands: []spec.Command{
			{Name: "dashboard", ID: "dashboard"},
			{Name: "session", ID: "session", Subcommands: []spec.Command{{Name: "start", ID: "session.start"}}},
		},
	}
	filtered := filterPekySpec(specDoc)
	if filtered.App.DefaultCommand != "" {
		t.Fatalf("default command = %q", filtered.App.DefaultCommand)
	}
	if len(filtered.Commands) != 1 || filtered.Commands[0].ID != "session" {
		t.Fatalf("filtered commands = %#v", filtered.Commands)
	}
	if len(filtered.Commands[0].Subcommands) != 1 || filtered.Commands[0].Subcommands[0].ID != "session.start" {
		t.Fatalf("filtered subcommands = %#v", filtered.Commands[0].Subcommands)
	}
}

func TestResolvePekyCommandID(t *testing.T) {
	specDoc := &spec.Spec{
		Commands: []spec.Command{
			{
				Name:    "session",
				ID:      "session",
				Aliases: []string{"sess"},
				Subcommands: []spec.Command{
					{Name: "start", ID: "session.start", Aliases: []string{"begin"}},
				},
			},
		},
	}
	id, err := resolvePekyCommandID(specDoc, []string{"sess", "begin"})
	if err != nil || id != "session.start" {
		t.Fatalf("resolve id = %q err=%v", id, err)
	}
	if _, err := resolvePekyCommandID(specDoc, []string{"unknown"}); err == nil {
		t.Fatalf("expected error for unknown command")
	}
}

func TestHasYesFlag(t *testing.T) {
	if hasYesFlag([]string{"run"}) {
		t.Fatalf("expected false without yes flag")
	}
	if !hasYesFlag([]string{"run", "--yes"}) {
		t.Fatalf("expected true for --yes")
	}
	if !hasYesFlag([]string{"run", "-y"}) {
		t.Fatalf("expected true for -y")
	}
}

func TestPekySuccessToast(t *testing.T) {
	if got := pekySuccessToast(""); got != "Done" {
		t.Fatalf("empty toast = %q", got)
	}
	if got := pekySuccessToast("  hello \n world  "); got != "hello world" {
		t.Fatalf("normalized toast = %q", got)
	}
	long := strings.Repeat("a", 140)
	got := pekySuccessToast(long)
	if len(got) != 120 || !strings.HasSuffix(got, "...") {
		t.Fatalf("long toast = %q", got)
	}
}

func TestHandlePekyPromptClear(t *testing.T) {
	m := newTestModelLite()
	m.pekyPromptLineID = 2
	m.pekyPromptLine = "ok"

	m.handlePekyPromptClear(pekyPromptClearMsg{ID: 1})
	if m.pekyPromptLine == "" {
		t.Fatalf("expected prompt line unchanged")
	}

	m.handlePekyPromptClear(pekyPromptClearMsg{ID: 2})
	if m.pekyPromptLine != "" {
		t.Fatalf("expected prompt line cleared")
	}
}
