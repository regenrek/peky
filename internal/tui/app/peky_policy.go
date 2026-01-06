package app

import (
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
)

type pekyPolicy struct {
	allowed []string
	blocked []string
}

func newPekyPolicy(cfg layout.AgentConfig) pekyPolicy {
	return pekyPolicy{
		allowed: cfg.AllowedCommands,
		blocked: cfg.BlockedCommands,
	}
}

func (p pekyPolicy) allows(commandID string) bool {
	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return false
	}
	if len(p.allowed) > 0 {
		return matchesAnyPattern(p.allowed, commandID)
	}
	if len(p.blocked) > 0 && matchesAnyPattern(p.blocked, commandID) {
		return false
	}
	return true
}

func matchesAnyPattern(patterns []string, commandID string) bool {
	for _, pattern := range patterns {
		if matchesCommandPattern(pattern, commandID) {
			return true
		}
	}
	return false
}

func matchesCommandPattern(pattern, commandID string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if pattern == commandID {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return commandID == prefix || strings.HasPrefix(commandID, prefix+".")
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(commandID, prefix)
	}
	if !strings.Contains(pattern, ".") {
		return commandID == pattern || strings.HasPrefix(commandID, pattern+".")
	}
	return false
}
