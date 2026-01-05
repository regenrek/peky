package layout

import "testing"

func TestApplyDefaultsAgent(t *testing.T) {
	cfg := Config{}
	ApplyDefaults(&cfg)
	if cfg.Agent.Provider != defaultAgentProvider {
		t.Fatalf("Agent.Provider=%q want %q", cfg.Agent.Provider, defaultAgentProvider)
	}
	if cfg.Agent.Model != defaultAgentModel {
		t.Fatalf("Agent.Model=%q want %q", cfg.Agent.Model, defaultAgentModel)
	}
	if len(cfg.Agent.BlockedCommands) == 0 {
		t.Fatalf("Agent.BlockedCommands should not be empty")
	}
}

func TestApplyDefaultsQuickReply(t *testing.T) {
	cfg := Config{}
	ApplyDefaults(&cfg)
	if cfg.QuickReply.Files.MaxDepth != defaultMaxDepth {
		t.Fatalf("QuickReply.Files.MaxDepth=%d want %d", cfg.QuickReply.Files.MaxDepth, defaultMaxDepth)
	}
	if cfg.QuickReply.Files.MaxItems != defaultMaxItems {
		t.Fatalf("QuickReply.Files.MaxItems=%d want %d", cfg.QuickReply.Files.MaxItems, defaultMaxItems)
	}
}

func TestApplyDefaultsDoesNotOverride(t *testing.T) {
	cfg := Config{
		Agent: AgentConfig{
			Provider:        "openai",
			Model:           "gpt-4o",
			BlockedCommands: []string{"pane.*"},
		},
		QuickReply: QuickReplyConfig{
			Files: QuickReplyFilesConfig{
				MaxDepth: 2,
				MaxItems: 25,
			},
		},
	}
	ApplyDefaults(&cfg)
	if cfg.Agent.Provider != "openai" {
		t.Fatalf("Agent.Provider=%q want %q", cfg.Agent.Provider, "openai")
	}
	if cfg.Agent.Model != "gpt-4o" {
		t.Fatalf("Agent.Model=%q want %q", cfg.Agent.Model, "gpt-4o")
	}
	if len(cfg.Agent.BlockedCommands) != 1 || cfg.Agent.BlockedCommands[0] != "pane.*" {
		t.Fatalf("Agent.BlockedCommands=%v", cfg.Agent.BlockedCommands)
	}
	if cfg.QuickReply.Files.MaxDepth != 2 || cfg.QuickReply.Files.MaxItems != 25 {
		t.Fatalf("QuickReply.Files=%+v", cfg.QuickReply.Files)
	}
}
