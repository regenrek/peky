package sessionrestore

import (
	"fmt"
	"strings"
)

// Mode controls how a pane participates in session restore persistence.
type Mode uint8

const (
	ModeDefault Mode = iota
	ModeEnabled
	ModeDisabled
	ModePrivate
)

func (m Mode) String() string {
	switch m {
	case ModeEnabled:
		return "enabled"
	case ModeDisabled:
		return "disabled"
	case ModePrivate:
		return "private"
	default:
		return "default"
	}
}

func (m Mode) AllowsPersistence(globalEnabled bool) bool {
	switch m {
	case ModeEnabled:
		return true
	case ModeDisabled:
		return false
	case ModePrivate:
		return false
	default:
		return globalEnabled
	}
}

func (m Mode) IsPrivate() bool {
	return m == ModePrivate
}

// ParseMode parses a mode string. Empty input returns ModeDefault.
func ParseMode(raw string) (Mode, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ModeDefault, nil
	}
	switch raw {
	case "true", "on", "yes", "enabled":
		return ModeEnabled, nil
	case "false", "off", "no", "disabled":
		return ModeDisabled, nil
	case "private":
		return ModePrivate, nil
	default:
		return ModeDefault, fmt.Errorf("invalid session restore mode %q", raw)
	}
}
