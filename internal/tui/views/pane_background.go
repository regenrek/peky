package views

import (
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func paneBackgroundOption(index int) (lipgloss.Color, bool) {
	if index < limits.PaneBackgroundMin || index > limits.PaneBackgroundMax {
		return "", false
	}
	options := theme.PaneBackgroundOptions
	if len(options) < limits.PaneBackgroundMax {
		return "", false
	}
	return options[index-1], true
}

func paneBackgroundTint(index int) (lipgloss.Color, bool) {
	if index == limits.PaneBackgroundDefault {
		return "", false
	}
	return paneBackgroundOption(index)
}

func paneBackgroundAnsiTint(index int) (xansi.Color, bool) {
	value, ok := paneBackgroundTint(index)
	if !ok {
		return nil, false
	}
	color := xansi.XParseColor(string(value))
	if color == nil {
		return nil, false
	}
	return color, true
}
