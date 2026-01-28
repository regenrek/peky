package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Target identifies a supported tool target.
type Target string

const (
	TargetCodex  Target = "codex"
	TargetClaude Target = "claude"
	TargetCursor Target = "cursor"
)

// Targets returns all supported targets in stable order.
func Targets() []Target {
	return []Target{TargetCodex, TargetClaude, TargetCursor}
}

// TargetLabel returns a human-friendly label for a target.
func TargetLabel(target Target) string {
	switch target {
	case TargetCodex:
		return "Codex CLI"
	case TargetClaude:
		return "Claude Code"
	case TargetCursor:
		return "Cursor"
	default:
		return strings.ToUpper(string(target))
	}
}

// ParseTargets parses a list of target strings (comma-separated or repeated).
func ParseTargets(values []string) ([]Target, error) {
	items := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.ToLower(strings.TrimSpace(item))
			if item == "" {
				continue
			}
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("target is required")
	}
	seen := make(map[Target]struct{})
	for _, item := range items {
		target, ok := targetFromString(item)
		if !ok {
			return nil, fmt.Errorf("unknown target %q", item)
		}
		seen[target] = struct{}{}
	}
	out := make([]Target, 0, len(seen))
	for target := range seen {
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func targetFromString(value string) (Target, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(TargetCodex):
		return TargetCodex, true
	case string(TargetClaude), "claude-code", "claude_code":
		return TargetClaude, true
	case string(TargetCursor):
		return TargetCursor, true
	default:
		return "", false
	}
}

// TargetRoot returns the default install root for a target.
func TargetRoot(target Target) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	switch target {
	case TargetCodex:
		return filepath.Join(home, ".codex", "skills"), nil
	case TargetClaude:
		return filepath.Join(home, ".claude", "skills"), nil
	case TargetCursor:
		return filepath.Join(home, ".cursor", "skills"), nil
	default:
		return "", fmt.Errorf("unknown target %q", target)
	}
}
