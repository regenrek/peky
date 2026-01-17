package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

func (m Model) viewFooter(width int) string {
	projectKeys := m.Keys.ProjectKeys
	sessionKeys := m.Keys.SessionKeys
	paneKeys := m.Keys.PaneKeys
	lastKey := m.Keys.ToggleLastPane
	actionKey := m.Keys.FocusAction
	commandsKey := m.Keys.CommandPalette
	rawKey := m.Keys.HardRaw
	helpKey := m.Keys.Help
	quitKey := m.Keys.Quit

	commonMods := commonChordMods([]string{
		projectKeys,
		sessionKeys,
		paneKeys,
		lastKey,
		actionKey,
		commandsKey,
		rawKey,
		helpKey,
		quitKey,
	})

	modPrefix := ""
	if len(commonMods) > 0 {
		modPrefix = strings.Join(commonMods, "+")
		projectKeys = stripChordMods(projectKeys, commonMods)
		sessionKeys = stripChordMods(sessionKeys, commonMods)
		paneKeys = stripChordMods(paneKeys, commonMods)
		lastKey = stripChordMods(lastKey, commonMods)
		actionKey = stripChordMods(actionKey, commonMods)
		commandsKey = stripChordMods(commandsKey, commonMods)
		rawKey = stripChordMods(rawKey, commonMods)
		helpKey = stripChordMods(helpKey, commonMods)
		quitKey = stripChordMods(quitKey, commonMods)
	}

	prefix := ""
	if modPrefix != "" {
		prefix = modPrefix + ": "
	}

	base := fmt.Sprintf(
		"%s%s project · %s session/pane · %s pane · %s last · %s action · %s commands · %s raw · %s help · %s quit",
		prefix,
		projectKeys,
		sessionKeys,
		paneKeys,
		lastKey,
		actionKey,
		commandsKey,
		rawKey,
		helpKey,
		quitKey,
	)
	base = theme.ListDimmed.Render(base)

	toast := m.Toast
	if toast == "" {
		return fitLineSuffix(base, m.viewFooterStatus(), width)
	}
	line := fmt.Sprintf("%s  %s", base, toast)
	return fitLineSuffix(line, m.viewFooterStatus(), width)
}

func (m Model) viewServerStatus() string {
	const slot = 8
	status := strings.ToLower(strings.TrimSpace(m.ServerStatus))
	switch status {
	case "down":
		word := theme.StatusError.Render("down")
		pad := slot - lipgloss.Width(word)
		if pad < 0 {
			pad = 0
		}
		return theme.ListDimmed.Render(strings.Repeat(" ", pad)) + word
	case "restored":
		word := theme.StatusWarning.Render("restored")
		pad := slot - lipgloss.Width(word)
		if pad < 0 {
			pad = 0
		}
		return theme.ListDimmed.Render(strings.Repeat(" ", pad)) + word
	default:
		word := theme.ListDimmed.Render("up")
		pad := slot - lipgloss.Width(word)
		if pad < 0 {
			pad = 0
		}
		return theme.ListDimmed.Render(strings.Repeat(" ", pad)) + word
	}
}

func (m Model) viewFooterStatus() string {
	return fmt.Sprintf("%s %s", m.viewInputBadge(), m.viewServerStatus())
}

func (m Model) viewInputBadge() string {
	const slot = 5
	word := "SOFT"
	style := theme.StatusMessage
	if m.HardRaw {
		word = "RAW"
		style = theme.StatusWarning
	}
	badge := style.Render(word)
	pad := slot - lipgloss.Width(badge)
	if pad < 0 {
		pad = 0
	}
	return theme.ListDimmed.Render(strings.Repeat(" ", pad)) + badge
}

func commonChordMods(labels []string) []string {
	var common map[string]struct{}
	for _, label := range labels {
		for _, chord := range splitChords(label) {
			mods := chordMods(chord)
			if len(mods) == 0 {
				return nil
			}
			if common == nil {
				common = make(map[string]struct{}, len(mods))
				for _, m := range mods {
					common[m] = struct{}{}
				}
				continue
			}
			next := make(map[string]struct{}, len(common))
			for _, m := range mods {
				if _, ok := common[m]; ok {
					next[m] = struct{}{}
				}
			}
			common = next
			if len(common) == 0 {
				return nil
			}
		}
	}
	return orderedMods(common)
}

func splitChords(label string) []string {
	parts := strings.Split(strings.TrimSpace(label), "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func chordMods(chord string) []string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(chord)), "+")
	if len(parts) <= 1 {
		return nil
	}
	mods := make([]string, 0, len(parts)-1)
	for _, p := range parts[:len(parts)-1] {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		mods = append(mods, p)
	}
	return mods
}

func orderedMods(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	order := []string{"ctrl", "alt", "shift", "meta"}
	out := make([]string, 0, len(order))
	for _, k := range order {
		if _, ok := set[k]; ok {
			out = append(out, k)
		}
	}
	return out
}

func stripChordMods(label string, common []string) string {
	commonSet := make(map[string]struct{}, len(common))
	for _, m := range common {
		commonSet[m] = struct{}{}
	}
	chords := splitChords(label)
	if len(chords) == 0 {
		return strings.TrimSpace(label)
	}
	out := make([]string, 0, len(chords))
	for _, chord := range chords {
		parts := strings.Split(strings.TrimSpace(chord), "+")
		if len(parts) == 0 {
			continue
		}
		base := strings.TrimSpace(parts[len(parts)-1])
		mods := make([]string, 0, len(parts)-1)
		for _, p := range parts[:len(parts)-1] {
			p = strings.ToLower(strings.TrimSpace(p))
			if p == "" {
				continue
			}
			if _, ok := commonSet[p]; ok {
				continue
			}
			mods = append(mods, p)
		}
		if len(mods) == 0 {
			out = append(out, base)
			continue
		}
		out = append(out, strings.Join(append(mods, base), "+"))
	}
	return strings.Join(out, "/")
}
