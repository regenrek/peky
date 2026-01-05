package layout

import "strings"

const (
	defaultAgentProvider = "google"
	defaultAgentModel    = "gemini-3-flash"
	defaultMaxDepth      = 4
	defaultMaxItems      = 500
)

var defaultBlockedCommands = []string{"daemon", "daemon.*", "pane.send"}

// ApplyDefaults fills in config defaults for agent and quick reply settings.
func ApplyDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	applyAgentDefaults(&cfg.Agent)
	applyQuickReplyDefaults(&cfg.QuickReply)
}

func applyAgentDefaults(cfg *AgentConfig) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.Provider) == "" {
		cfg.Provider = defaultAgentProvider
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultAgentModel
	}
	if len(cfg.BlockedCommands) == 0 {
		cfg.BlockedCommands = append([]string(nil), defaultBlockedCommands...)
	}
}

func applyQuickReplyDefaults(cfg *QuickReplyConfig) {
	if cfg == nil {
		return
	}
	if cfg.Files.MaxDepth <= 0 {
		cfg.Files.MaxDepth = defaultMaxDepth
	}
	if cfg.Files.MaxItems <= 0 {
		cfg.Files.MaxItems = defaultMaxItems
	}
}
