package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type quickReplyTarget struct {
	Pane PaneItem
}

type quickReplySendResult struct {
	ScopeLabel    string
	Total         int
	Sent          int
	Failed        int
	Skipped       int
	ClosedPaneIDs []string
	FirstError    string
}

type quickReplySendMsg struct {
	Result quickReplySendResult
}

func (m *Model) sendQuickReplyBroadcast(scope quickReplyScope, text string) tea.Cmd {
	message := strings.TrimSpace(text)
	if message == "" {
		return NewInfoCmd("Nothing to send")
	}
	if m.client == nil {
		return NewErrorCmd(errors.New("session client unavailable"), "send to panes")
	}
	targets, label := m.quickReplyBroadcastTargets(scope)
	if len(targets) == 0 {
		return NewWarningCmd("No panes to send to")
	}
	return func() tea.Msg {
		result := quickReplySendResult{
			ScopeLabel: label,
			Total:      len(targets),
		}
		for _, target := range targets {
			paneID := strings.TrimSpace(target.Pane.ID)
			if paneID == "" {
				result.Skipped++
				continue
			}
			if m.isPaneInputDisabled(paneID) {
				result.Skipped++
				continue
			}
			if pane := m.paneByID(paneID); pane == nil || pane.Dead {
				result.ClosedPaneIDs = append(result.ClosedPaneIDs, paneID)
				continue
			}
			payload := quickReplyInputBytes(target.Pane, message)
			logQuickReplySendAttempt(target.Pane, payload)
			ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
			err := m.client.SendInput(ctx, paneID, payload)
			cancel()
			if err != nil {
				logQuickReplySendError(paneID, err)
				if isPaneClosedError(err) {
					result.ClosedPaneIDs = append(result.ClosedPaneIDs, paneID)
					continue
				}
				result.Failed++
				if result.FirstError == "" {
					result.FirstError = err.Error()
				}
				continue
			}
			result.Sent++
		}
		return quickReplySendMsg{Result: result}
	}
}

func (m *Model) quickReplyBroadcastTargets(scope quickReplyScope) ([]quickReplyTarget, string) {
	switch scope {
	case quickReplyScopeProject:
		return m.quickReplyTargetsForProject()
	case quickReplyScopeAll:
		return m.quickReplyTargetsForAll()
	default:
		return m.quickReplyTargetsForSession()
	}
}

func (m *Model) quickReplyTargetsForSession() ([]quickReplyTarget, string) {
	session := m.selectedSession()
	if session == nil {
		return nil, "current session"
	}
	label := fmt.Sprintf("session %s", session.Name)
	return uniqueQuickReplyTargets(session.Panes), label
}

func (m *Model) quickReplyTargetsForProject() ([]quickReplyTarget, string) {
	project := m.selectedProject()
	if project == nil {
		return nil, "current project"
	}
	label := fmt.Sprintf("project %s", project.Name)
	var panes []PaneItem
	for _, session := range project.Sessions {
		panes = append(panes, session.Panes...)
	}
	return uniqueQuickReplyTargets(panes), label
}

func (m *Model) quickReplyTargetsForAll() ([]quickReplyTarget, string) {
	var panes []PaneItem
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			panes = append(panes, session.Panes...)
		}
	}
	return uniqueQuickReplyTargets(panes), "all panes"
}

func uniqueQuickReplyTargets(panes []PaneItem) []quickReplyTarget {
	seen := make(map[string]struct{})
	targets := make([]quickReplyTarget, 0, len(panes))
	for _, pane := range panes {
		paneID := strings.TrimSpace(pane.ID)
		if paneID == "" {
			continue
		}
		if _, ok := seen[paneID]; ok {
			continue
		}
		seen[paneID] = struct{}{}
		targets = append(targets, quickReplyTarget{Pane: pane})
	}
	return targets
}

func (m *Model) handleQuickReplySend(msg quickReplySendMsg) tea.Cmd {
	for _, paneID := range msg.Result.ClosedPaneIDs {
		m.markPaneInputDisabled(paneID)
	}
	message, level := quickReplySendSummary(msg.Result)
	if message != "" {
		m.setToast(message, level)
	}
	return nil
}

func quickReplySendSummary(result quickReplySendResult) (string, toastLevel) {
	scope := strings.TrimSpace(result.ScopeLabel)
	scopeSuffix := ""
	if scope != "" {
		scopeSuffix = " (" + scope + ")"
	}
	if result.Total == 0 {
		return "No panes to send to", toastInfo
	}
	if result.Sent == 0 {
		details := quickReplySendDetails(result)
		if details == "" {
			return "No panes accepted input" + scopeSuffix, toastWarning
		}
		return "No panes accepted input" + scopeSuffix + " — " + details, toastWarning
	}
	base := fmt.Sprintf("Sent to %d pane%s%s", result.Sent, pluralSuffix(result.Sent), scopeSuffix)
	details := quickReplySendDetails(result)
	if details == "" {
		return base, toastSuccess
	}
	return base + " — " + details, toastWarning
}

func quickReplySendDetails(result quickReplySendResult) string {
	var parts []string
	if result.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", result.Failed))
	}
	if len(result.ClosedPaneIDs) > 0 {
		parts = append(parts, fmt.Sprintf("%d closed", len(result.ClosedPaneIDs)))
	}
	if result.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", result.Skipped))
	}
	if result.FirstError != "" && result.Failed > 0 {
		parts = append(parts, "error: "+result.FirstError)
	}
	return strings.Join(parts, ", ")
}

func pluralSuffix(value int) string {
	if value == 1 {
		return ""
	}
	return "s"
}
