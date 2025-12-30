package termkeys

import "strings"

// IsCopyShortcutKey reports whether a key combo should trigger a copy yank.
func IsCopyShortcutKey(key string) bool {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return false
	}
	parts := strings.Split(key, "+")
	if len(parts) < 2 {
		return false
	}

	base := parts[len(parts)-1]
	mods := map[string]bool{}
	for _, raw := range parts[:len(parts)-1] {
		if raw == "" {
			continue
		}
		if mod := normalizeModifier(raw); mod != "" {
			mods[mod] = true
		}
	}

	switch base {
	case "c":
		if mods["alt"] {
			return false
		}
		return mods["ctrl"] || mods["cmd"] || mods["meta"]
	case "insert":
		if mods["shift"] || mods["alt"] || mods["cmd"] || mods["meta"] {
			return false
		}
		return mods["ctrl"]
	default:
		return false
	}
}

func normalizeModifier(value string) string {
	switch value {
	case "ctrl", "control":
		return "ctrl"
	case "cmd", "command":
		return "cmd"
	case "meta", "super":
		return "meta"
	case "alt", "option":
		return "alt"
	case "shift":
		return "shift"
	default:
		return ""
	}
}
