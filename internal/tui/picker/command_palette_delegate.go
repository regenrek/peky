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

	shortcutRendered, shortcutWidth, gapWidth := commandPaletteShortcut(shortcut, isSelected)
	titleStyle := commandPaletteTitleStyle(isSelected, &d.Styles)

	frameW, _ := titleStyle.GetFrameSize()
	textwidth := m.Width() - frameW - shortcutWidth - gapWidth
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

	title, desc = stylePaletteText(title, desc, titleStyle, &d.Styles, isSelected, isFiltered, matchedRunes)

	if shortcutRendered != "" {
		title = justifyTitleWithShortcut(title, shortcutRendered, m.Width())
	}
	if d.ShowDescription {
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

func commandPaletteShortcut(shortcut string, selected bool) (string, int, int) {
	if shortcut == "" {
		return "", 0, 0
	}
	shortcutStyle := theme.ShortcutHint
	if selected {
		shortcutStyle = theme.ShortcutDesc
	}
	rendered := shortcutStyle.Render(shortcut)
	return rendered, lipgloss.Width(rendered), 2
}

func commandPaletteTitleStyle(selected bool, styles *list.DefaultItemStyles) lipgloss.Style {
	if selected {
		return styles.SelectedTitle
	}
	return styles.NormalTitle
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

func stylePaletteText(
	title string,
	desc string,
	titleStyle lipgloss.Style,
	styles *list.DefaultItemStyles,
	selected bool,
	filtered bool,
	matchedRunes []int,
) (string, string) {
	if filtered {
		unmatched := titleStyle.Inline(true)
		matched := unmatched.Inherit(styles.FilterMatch)
		title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
	}
	title = titleStyle.Render(title)
	if selected {
		desc = styles.SelectedDesc.Render(desc)
	} else {
		desc = styles.NormalDesc.Render(desc)
	}
	return title, desc
}

func justifyTitleWithShortcut(title, shortcut string, width int) string {
	if width <= 0 {
		return title + " " + shortcut
	}
	titleWidth := lipgloss.Width(title)
	shortcutWidth := lipgloss.Width(shortcut)
	if titleWidth+shortcutWidth+1 > width {
		return title + " " + shortcut
	}
	gap := width - titleWidth - shortcutWidth
	if gap < 1 {
		gap = 1
	}
	return title + strings.Repeat(" ", gap) + shortcut
}
