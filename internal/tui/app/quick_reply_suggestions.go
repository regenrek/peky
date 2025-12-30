package app

import (
	"strings"
	"unicode"
)

type slashSuggestion struct {
	Text     string
	MatchLen int
	Desc     string
}

type slashInputContext struct {
	Active  bool
	Prefix  string
	HasArgs bool
}

func slashInputForSuggestions(input string) slashInputContext {
	if !strings.HasPrefix(input, "/") {
		return slashInputContext{}
	}
	body := input[1:]
	if body == "" {
		return slashInputContext{Active: true}
	}
	for i, r := range body {
		if unicode.IsSpace(r) {
			return slashInputContext{Active: true, Prefix: body[:i], HasArgs: true}
		}
	}
	return slashInputContext{Active: true, Prefix: body}
}

func (m *Model) slashSuggestions() []slashSuggestion {
	if m == nil || m.state != StateDashboard || m.filterActive || m.terminalFocus {
		return nil
	}
	ctx := slashInputForSuggestions(m.quickReplyInput.Value())
	if !ctx.Active || ctx.HasArgs {
		return nil
	}
	prefix := strings.ToLower(ctx.Prefix)
	entries, ok := m.slashSuggestionEntries(prefix, prefix != "")
	if !ok || len(entries) == 0 {
		return nil
	}
	matchLen := 1
	if prefix != "" {
		matchLen = len(prefix) + 1
	}
	suggestions := make([]slashSuggestion, 0, len(entries))
	for _, entry := range entries {
		text := "/" + entry.Alias
		current := matchLen
		if current > len(text) {
			current = len(text)
		}
		suggestions = append(suggestions, slashSuggestion{
			Text:     text,
			MatchLen: current,
			Desc:     entry.Desc,
		})
	}
	return suggestions
}

func (m *Model) applySlashCompletion() bool {
	ctx := slashInputForSuggestions(m.quickReplyInput.Value())
	if !ctx.Active || ctx.HasArgs {
		return false
	}
	if alias, ok := m.selectedSlashAlias(); ok {
		m.quickReplyInput.SetValue("/" + alias + " ")
		m.quickReplyInput.CursorEnd()
		return true
	}
	prefix := strings.ToLower(ctx.Prefix)
	matches, ok := m.slashCompletionMatches(prefix)
	if !ok || len(matches) == 0 {
		return false
	}
	if len(matches) == 1 {
		m.quickReplyInput.SetValue("/" + matches[0] + " ")
		m.quickReplyInput.CursorEnd()
		return true
	}
	if exactMatch(prefix, matches) {
		m.quickReplyInput.SetValue("/" + prefix + " ")
		m.quickReplyInput.CursorEnd()
		return true
	}
	common := longestCommonPrefix(matches)
	if common != "" && common != prefix {
		m.quickReplyInput.SetValue("/" + common)
		m.quickReplyInput.CursorEnd()
	}
	return true
}

func (m *Model) updateSlashSelection() {
	if m == nil {
		return
	}
	ctx := slashInputForSuggestions(m.quickReplyInput.Value())
	if !ctx.Active || ctx.HasArgs || m.state != StateDashboard || m.filterActive || m.terminalFocus {
		m.resetSlashMenu()
		return
	}
	prefix := strings.ToLower(ctx.Prefix)
	if prefix != m.quickReplySlashPrefix {
		m.quickReplySlashPrefix = prefix
		m.quickReplySlashIndex = -1
	}
	suggestions := m.slashSuggestions()
	if len(suggestions) == 0 {
		m.quickReplySlashIndex = -1
		return
	}
	if m.quickReplySlashIndex < 0 || m.quickReplySlashIndex >= len(suggestions) {
		m.quickReplySlashIndex = 0
	}
}

func (m *Model) moveSlashSelection(delta int) bool {
	if m == nil {
		return false
	}
	m.updateSlashSelection()
	suggestions := m.slashSuggestions()
	if len(suggestions) == 0 {
		return false
	}
	if m.quickReplySlashIndex < 0 || m.quickReplySlashIndex >= len(suggestions) {
		m.quickReplySlashIndex = 0
	}
	count := len(suggestions)
	next := (m.quickReplySlashIndex + delta) % count
	if next < 0 {
		next += count
	}
	m.quickReplySlashIndex = next
	return true
}

func (m *Model) selectedSlashAlias() (string, bool) {
	if m == nil {
		return "", false
	}
	suggestions := m.slashSuggestions()
	if len(suggestions) == 0 {
		return "", false
	}
	if m.quickReplySlashIndex < 0 || m.quickReplySlashIndex >= len(suggestions) {
		return "", false
	}
	alias := strings.TrimPrefix(suggestions[m.quickReplySlashIndex].Text, "/")
	if alias == "" {
		return "", false
	}
	return alias, true
}

func (m *Model) resetSlashMenu() {
	m.quickReplySlashIndex = -1
	m.quickReplySlashPrefix = ""
}

type slashSuggestionEntry struct {
	Alias string
	Desc  string
}

func (m *Model) slashCompletionMatches(prefix string) ([]string, bool) {
	entries, ok := m.slashSuggestionEntries(prefix, true)
	if !ok {
		return nil, false
	}
	matches := make([]string, 0, len(entries))
	for _, entry := range entries {
		matches = append(matches, entry.Alias)
	}
	return matches, true
}

func (m *Model) slashSuggestionEntries(prefix string, includeAliases bool) ([]slashSuggestionEntry, bool) {
	registry, err := m.commandRegistry()
	if err != nil {
		return nil, false
	}
	entries := make([]slashSuggestionEntry, 0, 16)
	seen := make(map[string]struct{})
	addEntry := func(alias, desc string) {
		normalized := normalizeCommandAlias(alias)
		if normalized == "" {
			return
		}
		if prefix != "" && !strings.HasPrefix(normalized, prefix) {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		entries = append(entries, slashSuggestionEntry{Alias: normalized, Desc: desc})
	}

	broadcastAdded := false
	for _, group := range registry.Groups {
		if group.Name == "broadcast" && !broadcastAdded {
			desc := ""
			if len(group.Commands) > 0 {
				desc = strings.TrimSpace(group.Commands[0].Desc)
			}
			if desc == "" {
				desc = "Send to session/project/all panes"
			}
			addEntry("all", desc)
			broadcastAdded = true
		}
		for _, cmd := range group.Commands {
			desc := strings.TrimSpace(cmd.Desc)
			if desc == "" {
				desc = strings.TrimSpace(cmd.Label)
			}
			if includeAliases {
				for _, alias := range cmd.Aliases {
					addEntry(alias, desc)
				}
				continue
			}
			if len(cmd.Aliases) == 0 {
				continue
			}
			addEntry(cmd.Aliases[0], desc)
		}
	}
	if !broadcastAdded {
		addEntry("all", "Send to session/project/all panes")
	}
	return entries, true
}

func longestCommonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := values[0]
	for _, value := range values[1:] {
		for prefix != "" && !strings.HasPrefix(value, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			return ""
		}
	}
	return prefix
}

func exactMatch(prefix string, values []string) bool {
	if prefix == "" {
		return false
	}
	for _, value := range values {
		if value == prefix {
			return true
		}
	}
	return false
}
