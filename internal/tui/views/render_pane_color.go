package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/limits"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewPaneColor() string {
	if !m.PaneColor.Open {
		return ""
	}
	heading := dialogTitleStyle.Render("Pane Color")
	details := paneColorDetails(m.PaneColor)
	palette := paneColorPalette(m.PaneColor.Current)
	hint := theme.DialogNote.Render("Type 1-5 to set, esc to close")
	content := dialogContent(heading, details, palette, hint)
	return m.renderDialog(dialogSpec{Content: content, RequireViewport: true})
}

func paneColorDetails(dialog PaneColorDialog) string {
	pane := strings.TrimSpace(dialog.PaneLabel)
	session := strings.TrimSpace(dialog.Session)
	lines := make([]string, 0, 2)
	if pane != "" {
		lines = append(lines, theme.DialogLabel.Render("Pane:")+" "+theme.DialogValue.Render(pane))
	}
	if session != "" {
		lines = append(lines, theme.DialogLabel.Render("Session:")+" "+theme.DialogValue.Render(session))
	}
	return strings.Join(lines, "\n")
}

func paneColorPalette(current int) string {
	names := []string{"default", "blue", "green", "amber", "violet"}
	lines := make([]string, 0, limits.PaneBackgroundMax)
	for i := limits.PaneBackgroundMin; i <= limits.PaneBackgroundMax; i++ {
		color, ok := paneBackgroundOption(i)
		if !ok {
			color = theme.Background
		}
		square := lipgloss.NewStyle().Background(color).Render("  ")
		label := "color"
		if i-1 < len(names) {
			label = names[i-1]
		}
		key := fmt.Sprintf("%d", i)
		if i == current {
			key = theme.DialogChoiceKey.Render(key)
		}
		line := fmt.Sprintf("%s %s %s", key, square, label)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
