package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"

	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

func matchesBinding(msg tuiinput.KeyMsg, binding key.Binding) bool {
	needle := strings.TrimSpace(strings.ToLower(msg.Keystroke()))
	if needle == "" {
		return false
	}
	for _, k := range binding.Keys() {
		if needle == strings.ToLower(strings.TrimSpace(k)) {
			return true
		}
	}
	return false
}
