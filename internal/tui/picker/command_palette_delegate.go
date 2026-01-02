package picker

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

const commandPaletteEllipsis = "…"

type commandPaletteDelegate struct {
	list.DefaultDelegate
}

func newCommandPaletteDelegate() commandPaletteDelegate {
	return commandPaletteDelegate{DefaultDelegate: list.NewDefaultDelegate()}
}

func (d commandPaletteDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	title, desc, shortcut, ok := commandPaletteItem(item)
	if !ok || m.Width() <= 0 {
		return
	}

	isSelected := index == m.Index()
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied

	rowStyle := commandPaletteRowStyle(isSelected, m.Width())
	innerWidth := commandPaletteInnerWidth(rowStyle, m.Width())
	shortcutWidth := lipgloss.Width(shortcut)
	gapWidth := 0
	if shortcutWidth > 0 {
		gapWidth = 1
	}
	textwidth := innerWidth - shortcutWidth - gapWidth
	if textwidth < 0 {
		textwidth = 0
	}
	title = ansi.Truncate(title, textwidth, commandPaletteEllipsis)
	if d.ShowDescription {
		desc = truncatePaletteDescription(desc, textwidth, d.Height())
	}

	var matchedRunes []int
	if isFiltered {
		matchedRunes = m.MatchesForItem(index)
	}

	rowTextStyle, rowMatchStyle, rowShortcutStyle, rowGapStyle := commandPaletteTextStyles(isSelected)
	if isFiltered {
		title = lipgloss.StyleRunes(title, matchedRunes, rowMatchStyle, rowTextStyle)
	} else {
		title = rowTextStyle.Render(title)
	}
	contentWidth := lipgloss.Width(title)
	if shortcutWidth > 0 {
		gap := innerWidth - lipgloss.Width(title) - shortcutWidth
		if gap < 1 {
			gap = 1
		}
		title = title + rowGapStyle.Render(strings.Repeat(" ", gap)) + rowShortcutStyle.Render(shortcut)
		contentWidth = lipgloss.Width(title)
	}
	if innerWidth > contentWidth {
		title += rowGapStyle.Render(strings.Repeat(" ", innerWidth-contentWidth))
	}

	title = rowStyle.Render(title)
	if d.ShowDescription {
		descStyle := commandPaletteDescStyle(isSelected)
		if desc != "" {
			desc = descStyle.Render(desc)
			desc = rowStyle.Render(desc)
		}
		fmt.Fprintf(w, "%s\n%s", title, desc) //nolint: errcheck
		return
	}
	fmt.Fprintf(w, "%s", title) //nolint: errcheck
}

func commandPaletteItem(item list.Item) (string, string, string, bool) {
	i, ok := item.(list.DefaultItem)
	if !ok {
		return "", "", "", false
	}
	title := i.Title()
	desc := i.Description()
	shortcut := ""
	if cmd, ok := item.(CommandItem); ok {
		shortcut = strings.TrimSpace(cmd.Shortcut)
	}
	return title, desc, shortcut, true
}

func commandPaletteRowStyle(selected bool, width int) lipgloss.Style {
	rowStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(width).
		Background(theme.Background).
		BorderLeft(true).
		BorderLeftForeground(theme.Background)
	if selected {
		rowStyle = rowStyle.
			Background(theme.AccentFocus).
			BorderLeftForeground(theme.AccentAlt)
	}
	return rowStyle
}

func commandPaletteInnerWidth(style lipgloss.Style, width int) int {
	frameW, _ := style.GetFrameSize()
	inner := width - frameW
	if inner < 0 {
		return 0
	}
	return inner
}

func commandPaletteTextStyles(selected bool) (lipgloss.Style, lipgloss.Style, lipgloss.Style, lipgloss.Style) {
	rowBg := theme.Background
	if selected {
		rowBg = theme.AccentFocus
	}
	textStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary).
		Background(rowBg)
	if selected {
		textStyle = textStyle.
			Foreground(theme.Surface).
			Bold(true)
	}
	matchStyle := textStyle.Copy().Bold(true)
	if selected {
		matchStyle = matchStyle.Foreground(theme.Surface)
	} else {
		matchStyle = matchStyle.Foreground(theme.AccentFocus)
	}
	shortcutStyle := theme.ShortcutHint.Background(rowBg)
	if selected {
		shortcutStyle = theme.ShortcutDesc.
			Foreground(theme.Surface).
			Background(rowBg)
	}
	gapStyle := lipgloss.NewStyle().Background(rowBg)
	return textStyle, matchStyle, shortcutStyle, gapStyle
}

func commandPaletteDescStyle(selected bool) lipgloss.Style {
	if selected {
		return lipgloss.NewStyle().Foreground(theme.TextSecondary)
	}
	return lipgloss.NewStyle().Foreground(theme.TextDim)
}

func truncatePaletteDescription(desc string, textwidth int, height int) string {
	if height <= 1 {
		return ""
	}
	lines := make([]string, 0, height-1)
	for i, line := range strings.Split(desc, "\n") {
		if i >= height-1 {
			break
		}
		lines = append(lines, ansi.Truncate(line, textwidth, commandPaletteEllipsis))
	}
	return strings.Join(lines, "\n")
}
