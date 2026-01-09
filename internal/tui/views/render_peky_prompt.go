package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewPekyPromptLine(width int) string {
	text := strings.TrimSpace(m.PekyPromptLine)
	if text == "" || width <= 0 {
		return ""
	}
	barWidth := width
	contentWidth := barWidth - 4
	if contentWidth < 10 {
		contentWidth = 10
	}
	base := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyTag)
	label := base.Foreground(theme.Accent).Bold(true).Render("peky ")
	flat := strings.ReplaceAll(text, "\n", " ")
	flat = strings.Join(strings.Fields(flat), " ")
	line := label + base.Render(flat)
	line = ansi.Truncate(line, contentWidth, "")
	visible := lipgloss.Width(line)
	if visible < contentWidth {
		line += base.Render(strings.Repeat(" ", contentWidth-visible))
	}
	pad := base.Render(strings.Repeat(" ", 2))
	return pad + line + pad
}
