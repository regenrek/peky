package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
