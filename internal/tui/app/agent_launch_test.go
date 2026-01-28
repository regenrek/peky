package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestResolveAgentLaunchDefaults(t *testing.T) {
	m := newTestModelLite()

	cmd, title, err := m.resolveAgentLaunch(agentLaunchDefs[0])
	if err != nil {
		t.Fatalf("resolveAgentLaunch error: %v", err)
	}
	if cmd != "codex" {
		t.Fatalf("cmd=%q want %q", cmd, "codex")
	}
	if title != "codex" {
		t.Fatalf("title=%q want %q", title, "codex")
	}
}

func TestResolveAgentLaunchProjectOverride(t *testing.T) {
	m := newTestModelLite()
	m.config.Tools = layout.ToolsConfig{
		CodexCLI: layout.ToolConfig{
			Cmd:       "codex --global",
			PaneTitle: "global",
		},
	}

	projectPath := normalizeProjectPath("/alpha")
	m.projectLocalConfig = map[string]projectLocalConfigCache{
		projectPath: {
			config: &layout.ProjectLocalConfig{
				Tools: layout.ToolsConfig{
					CodexCLI: layout.ToolConfig{
						Cmd: "codex --local",
					},
				},
			},
		},
	}

	cmd, title, err := m.resolveAgentLaunch(agentLaunchDefs[0])
	if err != nil {
		t.Fatalf("resolveAgentLaunch error: %v", err)
	}
	if cmd != "codex --local" {
		t.Fatalf("cmd=%q want %q", cmd, "codex --local")
	}
	if title != "global" {
		t.Fatalf("title=%q want %q", title, "global")
	}
}

func TestValidateAgentCommand(t *testing.T) {
	if _, err := validateAgentCommand("co\ndex"); err == nil {
		t.Fatalf("expected validation error")
	}
}
