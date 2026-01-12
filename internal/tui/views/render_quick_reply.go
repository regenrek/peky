package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewQuickReply(width int) string {
	if width <= 0 {
		return ""
	}
	barWidth := width
	contentWidth := barWidth - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	base := lipgloss.NewStyle().
		Foreground(theme.TextPrimary).
		Background(theme.QuickReplyBg)
	accent := base.Foreground(theme.Accent).Render("▌ ")
	label := ""
	labelWidth := 0
	if mode := strings.TrimSpace(m.QuickReplyMode); mode != "" {
		label = base.Foreground(theme.Accent).Bold(true).Render(mode) + base.Render(" ")
		labelWidth = lipgloss.Width(label)
	}
	inputWidth := contentWidth - lipgloss.Width(accent) - labelWidth
	if inputWidth < 10 {
		inputWidth = 10
	}
	if inputWidth > contentWidth {
		inputWidth = contentWidth
	}
	m.QuickReplyInput.Width = inputWidth

	inputView := m.QuickReplyInput.View()
	if m.QuickReplySelectionActive {
		if v := renderQuickReplyInputSelection(base, m.QuickReplyInput.Value(), m.QuickReplySelectionStart, m.QuickReplySelectionEnd); v != "" {
			inputView = v
		}
	}

	line := accent + label + inputView
	line = ansi.Truncate(line, contentWidth, "")
	visible := lipgloss.Width(line)
	if visible < contentWidth {
		line += base.Render(strings.Repeat(" ", contentWidth-visible))
	}

	pad := base.Render(strings.Repeat(" ", 2))
	blank := base.Render(strings.Repeat(" ", contentWidth))
	lines := []string{
		pad + blank + pad,
		pad + line + pad,
	}
	lines = append(lines, pad+blank+pad)
	return strings.Join(lines, "\n")
}

func renderQuickReplyInputSelection(base lipgloss.Style, value string, start, end int) string {
	value = strings.TrimRight(value, "\n")
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > len(runes) {
		start = len(runes)
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start > end {
		start, end = end, start
	}
	if start == end {
		return ""
	}

	selected := base.Reverse(true).Bold(true)
	return base.Render(string(runes[:start])) + selected.Render(string(runes[start:end])) + base.Render(string(runes[end:]))
}

func (m Model) overlayQuickReplyMenu(base string, width, height, headerHeight, headerGap, bodyHeight int) string {
	if len(m.QuickReplySuggestions) == 0 || width <= 0 || height <= 0 {
		return base
	}
	menuX := 2
	menuWidth := width - 4
	if menuWidth < 10 {
		menuWidth = width
		menuX = 0
	}
	availableHeight := headerHeight + headerGap + bodyHeight
	menu := renderQuickReplyMenu(m.QuickReplySuggestions, m.QuickReplySelected, menuWidth, availableHeight)
	if strings.TrimSpace(menu) == "" {
		return base
	}
	menuHeight := lipgloss.Height(menu)
	if menuHeight <= 0 {
		return base
	}
	menuY := availableHeight - menuHeight
	if menuY < 0 {
		menuY = 0
	}
	return overlayAt(base, menu, width, height, menuX, menuY)
}
