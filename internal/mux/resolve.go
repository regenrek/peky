package mux

import (
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
)

// ResolveType chooses the multiplexer based on CLI override and config precedence.
func ResolveType(cliValue string, cfg *layout.Config, project *layout.ProjectConfig, local *layout.ProjectLocalConfig) Type {
	if t, ok := parseIfSet(cliValue); ok {
		return t
	}
	if local != nil {
		if t, ok := parseIfSet(local.Multiplexer); ok {
			return t
		}
	}
	if project != nil {
		if t, ok := parseIfSet(project.Multiplexer); ok {
			return t
		}
	}
	if cfg != nil {
		if t, ok := parseIfSet(cfg.Multiplexer); ok {
			return t
		}
	}
	return Tmux
}

func parseIfSet(value string) (Type, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	t, err := ParseType(value)
	if err != nil {
		return "", false
	}
	return t, true
}
