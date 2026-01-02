package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kballard/go-shellquote"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

type quickReplyScope string

const (
	quickReplyScopeSession quickReplyScope = "session"
	quickReplyScopeProject quickReplyScope = "project"
	quickReplyScopeAll     quickReplyScope = "all"
)

type quickReplyCommandOutcome struct {
	Cmd          tea.Cmd
	Handled      bool
	ClearInput   bool
	RecordPrompt bool
}

func (m *Model) handleQuickReplyCommand(input string) quickReplyCommandOutcome {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return quickReplyCommandOutcome{}
	}
	if scope, text, ok := parseQuickReplyBroadcast(trimmed); ok {
		if strings.TrimSpace(text) == "" {
			return quickReplyCommandOutcome{
				Cmd:     NewInfoCmd("Nothing to send"),
				Handled: true,
			}
		}
		return quickReplyCommandOutcome{
			Cmd:          m.sendQuickReplyBroadcast(scope, text),
			Handled:      true,
			ClearInput:   true,
			RecordPrompt: true,
		}
	}
	if !strings.HasPrefix(trimmed, "/") {
		return quickReplyCommandOutcome{}
	}
	cmd, matched, clear, record := m.runSlashCommand(trimmed)
	return quickReplyCommandOutcome{
		Cmd:          cmd,
		Handled:      true,
		ClearInput:   clear,
		RecordPrompt: record && matched,
	}
}

func (m *Model) runSlashCommand(input string) (tea.Cmd, bool, bool, bool) {
	body := strings.TrimSpace(strings.TrimPrefix(input, "/"))
	if body == "" {
		return NewInfoCmd("Enter a command"), true, false, false
	}
	registry, err := m.commandRegistry()
	if err != nil {
		return NewErrorCmd(err, "command registry"), true, false, false
	}
	tokens := strings.Fields(body)
	if len(tokens) == 0 {
		return NewInfoCmd("Enter a command"), true, false, false
	}
	lowerTokens := make([]string, len(tokens))
	for i, token := range tokens {
		lowerTokens[i] = strings.ToLower(token)
	}
	for _, alias := range registry.Aliases {
		if len(lowerTokens) < len(alias.tokens) {
			continue
		}
		if !tokensMatch(lowerTokens[:len(alias.tokens)], alias.tokens) {
			continue
		}
		argsTokens := tokens[len(alias.tokens):]
		args := commandArgs{
			Raw:    strings.TrimSpace(strings.Join(argsTokens, " ")),
			Tokens: argsTokens,
		}
		if alias.command.Run == nil {
			return nil, true, true, true
		}
		return alias.command.Run(m, args), true, true, true
	}
	if cmd, ok, clear, record := m.runSlashShortcut(body); ok {
		return cmd, true, clear, record
	}
	return NewWarningCmd("Unknown command"), false, false, false
}

func parseQuickReplyBroadcast(input string) (quickReplyScope, string, bool) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "/all") {
		return "", "", false
	}
	if len(trimmed) > 4 {
		next := trimmed[4:5]
		if next != " " && next != "\t" {
			return "", "", false
		}
	}
	rest := strings.TrimSpace(trimmed[4:])
	scope := quickReplyScopeSession
	if rest == "" {
		return scope, "", true
	}
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return scope, "", true
	}
	switch strings.ToLower(parts[0]) {
	case "session":
		scope = quickReplyScopeSession
		rest = strings.TrimSpace(rest[len(parts[0]):])
	case "project":
		scope = quickReplyScopeProject
		rest = strings.TrimSpace(rest[len(parts[0]):])
	case "all", "global":
		scope = quickReplyScopeAll
		rest = strings.TrimSpace(rest[len(parts[0]):])
	}
	return scope, strings.TrimSpace(rest), true
}

func tokensMatch(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func (m *Model) runSlashShortcut(body string) (tea.Cmd, bool, bool, bool) {
	parts, ok := parseSlashShortcutParts(body)
	if !ok {
		return nil, false, false, false
	}
	shortcut, ok := loadSlashShortcut(parts[0])
	if !ok {
		return nil, false, false, false
	}
	text := strings.TrimSpace(strings.Join(parts[1:], " "))
	return m.executeSlashShortcut(shortcut, text)
}

func parseSlashShortcutParts(body string) ([]string, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, false
	}
	parts, err := shellquote.Split(body)
	if err != nil || len(parts) == 0 {
		return nil, false
	}
	return parts, true
}

func loadSlashShortcut(name string) (spec.SlashShortcut, bool) {
	doc, err := spec.LoadDefault()
	if err != nil || doc == nil {
		return spec.SlashShortcut{}, false
	}
	name = strings.ToLower(name)
	for _, shortcut := range doc.SlashShortcuts {
		if strings.ToLower(shortcut.Name) == name {
			return shortcut, true
		}
	}
	return spec.SlashShortcut{}, false
}

func (m *Model) executeSlashShortcut(shortcut spec.SlashShortcut, text string) (tea.Cmd, bool, bool, bool) {
	command := strings.TrimSpace(shortcut.Command)
	if command == "" {
		return NewWarningCmd("Unknown command"), true, true, false
	}
	if text == "" {
		return NewInfoCmd("Nothing to send"), true, false, false
	}
	scope := shortcutScope(shortcut)
	if paneID, ok := shortcutPaneID(shortcut); ok {
		if pane := m.paneByID(paneID); pane != nil {
			return m.sendQuickReplySingle(*pane, text, shortcutDelay(shortcut)), true, true, true
		}
	}
	if command == "pane send" || command == "pane run" {
		return m.sendQuickReplyBroadcastWithDelay(scope, text, shortcutDelay(shortcut)), true, true, true
	}
	return NewWarningCmd("Unsupported slash shortcut"), true, false, false
}

func shortcutScope(shortcut spec.SlashShortcut) quickReplyScope {
	scope := quickReplyScopeSession
	if raw, ok := shortcut.Flags["scope"]; ok {
		if value, ok := raw.(string); ok {
			scope = parseQuickReplyScope(value)
		}
	}
	return scope
}

func shortcutPaneID(shortcut spec.SlashShortcut) (string, bool) {
	raw, ok := shortcut.Flags["pane-id"]
	if !ok {
		return "", false
	}
	id, ok := raw.(string)
	if !ok || strings.TrimSpace(id) == "" {
		return "", false
	}
	return id, true
}

func shortcutDelay(shortcut spec.SlashShortcut) time.Duration {
	if raw, ok := shortcut.Flags["delay"]; ok {
		if value, ok := raw.(string); ok {
			if d, err := time.ParseDuration(value); err == nil {
				return d
			}
		}
	}
	return 0
}

func parseQuickReplyScope(value string) quickReplyScope {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "project":
		return quickReplyScopeProject
	case "all":
		return quickReplyScopeAll
	default:
		return quickReplyScopeSession
	}
}

func (m *Model) sendQuickReplyBroadcastWithDelay(scope quickReplyScope, text string, delay time.Duration) tea.Cmd {
	cmd := m.sendQuickReplyBroadcast(scope, text)
	if delay <= 0 {
		return cmd
	}
	return func() tea.Msg {
		time.Sleep(delay)
		if cmd == nil {
			return nil
		}
		return cmd()
	}
}

func (m *Model) sendQuickReplySingle(pane PaneItem, text string, delay time.Duration) tea.Cmd {
	if strings.TrimSpace(text) == "" {
		return NewInfoCmd("Nothing to send")
	}
	return func() tea.Msg {
		if delay > 0 {
			time.Sleep(delay)
		}
		result := quickReplySendResult{ScopeLabel: pane.Title, Total: 1}
		target := quickReplyTarget{Pane: pane}
		tool := quickReplyToolFromText(text)
		targetResult := m.sendQuickReplyToTarget(target, text, tool)
		applyQuickReplyTargetResult(&result, targetResult)
		return quickReplySendMsg{Result: result}
	}
}
