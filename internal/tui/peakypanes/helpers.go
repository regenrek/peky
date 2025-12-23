package peakypanes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

func expandPath(p string) string {
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			if strings.HasPrefix(p, "~/") {
				return filepath.Join(home, p[2:])
			}
		}
	}
	return p
}

func shortenPath(p string) string {
	if p == "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func sanitizeSessionName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "session"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "session"
	}
	return result
}

func validateTmuxName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("Name cannot be empty")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("Name contains control characters")
		}
	}
	return nil
}

func selfExecutable() string {
	exe, err := os.Executable()
	if err != nil || strings.TrimSpace(exe) == "" {
		return "peakypanes"
	}
	return exe
}

func validateProjectPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func nextSessionName(base string, existing []string) string {
	base = sanitizeSessionName(base)
	if base == "" {
		base = "session"
	}
	used := make(map[string]struct{}, len(existing))
	for _, name := range existing {
		used[name] = struct{}{}
	}
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}
