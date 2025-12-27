package layout

import "strings"

const (
	MultiplexerNative = "native"
	MultiplexerTmux   = "tmux"
)

// NormalizeMultiplexer returns the canonical multiplexer name.
// Empty values return empty string so callers can continue resolution.
// Any non-empty value other than "tmux" normalizes to "native".
func NormalizeMultiplexer(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(trimmed, MultiplexerTmux) {
		return MultiplexerTmux
	}
	return MultiplexerNative
}

// ResolveMultiplexer resolves the multiplexer in precedence order:
// project-local config, global project entry, global config root, default native.
func ResolveMultiplexer(local *ProjectLocalConfig, project *ProjectConfig, cfg *Config) string {
	if local != nil {
		if resolved := NormalizeMultiplexer(local.Multiplexer); resolved != "" {
			return resolved
		}
	}
	if project != nil {
		if resolved := NormalizeMultiplexer(project.Multiplexer); resolved != "" {
			return resolved
		}
	}
	if cfg != nil {
		if resolved := NormalizeMultiplexer(cfg.Multiplexer); resolved != "" {
			return resolved
		}
	}
	return MultiplexerNative
}
