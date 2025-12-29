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
	var (
		title, desc  string
		shortcut     string
		matchedRunes []int
		s            = &d.Styles
	)

	if i, ok := item.(list.DefaultItem); ok {
		title = i.Title()
		desc = i.Description()
		if cmd, ok := item.(CommandItem); ok {
			shortcut = strings.TrimSpace(cmd.Shortcut)
		}
	} else {
		return
	}

	if m.Width() <= 0 {
		return
	}

	shortcutStyle := theme.ShortcutHint
	if shortcut != "" && index == m.Index() {
		shortcutStyle = theme.ShortcutDesc
	}
	shortcutRendered := ""
	shortcutWidth := 0
	gapWidth := 0
	if shortcut != "" {
		shortcutRendered = shortcutStyle.Render(shortcut)
		shortcutWidth = lipgloss.Width(shortcutRendered)
		gapWidth = 2
	}

	var titleStyle lipgloss.Style
	isSelected := index == m.Index()
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied

	switch {
	case isSelected:
		titleStyle = s.SelectedTitle
	default:
		titleStyle = s.NormalTitle
	}

	frameW, _ := titleStyle.GetFrameSize()
	textwidth := m.Width() - frameW - shortcutWidth - gapWidth
	if textwidth < 0 {
		textwidth = 0
	}
	title = ansi.Truncate(title, textwidth, commandPaletteEllipsis)
	if d.ShowDescription {
		var lines []string
		for i, line := range strings.Split(desc, "\n") {
			if i >= d.Height()-1 {
				break
			}
			lines = append(lines, ansi.Truncate(line, textwidth, commandPaletteEllipsis))
		}
		desc = strings.Join(lines, "\n")
	}

	if isFiltered {
		matchedRunes = m.MatchesForItem(index)
	}

	if isSelected {
		if isFiltered {
			unmatched := titleStyle.Inline(true)
			matched := unmatched.Inherit(s.FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
		title = titleStyle.Render(title)
		desc = s.SelectedDesc.Render(desc)
	} else {
		if isFiltered {
			unmatched := titleStyle.Inline(true)
			matched := unmatched.Inherit(s.FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
		title = titleStyle.Render(title)
		desc = s.NormalDesc.Render(desc)
	}

	if d.ShowDescription {
		if shortcutRendered != "" {
			title = justifyTitleWithShortcut(title, shortcutRendered, m.Width())
		}
		fmt.Fprintf(w, "%s\n%s", title, desc) //nolint: errcheck
		return
	}
	if shortcutRendered != "" {
		title = justifyTitleWithShortcut(title, shortcutRendered, m.Width())
	}
	fmt.Fprintf(w, "%s", title) //nolint: errcheck
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
