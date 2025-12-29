package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// ErrorMsg is a tea.Msg for error handling in the TUI.
// Following best practices: errors are surfaced as messages, not inline.
type ErrorMsg struct {
	Err     error
	Context string // e.g., "loading config", "killing session"
}

func (e ErrorMsg) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %v", e.Context, e.Err)
	}
	return e.Err.Error()
}

// NewErrorMsg creates an ErrorMsg with context.
func NewErrorMsg(err error, context string) ErrorMsg {
	return ErrorMsg{Err: err, Context: context}
}

// SuccessMsg is a tea.Msg for success notifications.
type SuccessMsg struct {
	Message string
}

// InfoMsg is a tea.Msg for informational notifications.
type InfoMsg struct {
	Message string
}

// WarningMsg is a tea.Msg for warning notifications.
type WarningMsg struct {
	Message string
}

// RefreshDoneMsg signals that a refresh operation completed.
type RefreshDoneMsg struct {
	Err error
}

// SessionKilledMsg signals a session was killed.
type SessionKilledMsg struct {
	Session string
	Err     error
}

// SessionStartedMsg signals a session was started.
type SessionStartedMsg struct {
	Session string
	Err     error
}

// ===== Commands =====

// NewErrorCmd creates a command that sends an ErrorMsg.
func NewErrorCmd(err error, context string) tea.Cmd {
	return func() tea.Msg {
		return NewErrorMsg(err, context)
	}
}

// NewSuccessCmd creates a command that sends a SuccessMsg.
func NewSuccessCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return SuccessMsg{Message: message}
	}
}

// NewInfoCmd creates a command that sends an InfoMsg.
func NewInfoCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return InfoMsg{Message: message}
	}
}

// NewWarningCmd creates a command that sends a WarningMsg.
func NewWarningCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return WarningMsg{Message: message}
	}
}

// ===== Helper functions for status bar messages =====

// FormatStatusError formats an error for the status bar.
func FormatStatusError(err error) string {
	return theme.FormatError(err.Error())
}

// FormatStatusSuccess formats a success message for the status bar.
func FormatStatusSuccess(msg string) string {
	return theme.FormatSuccess(msg)
}

// FormatStatusWarning formats a warning message for the status bar.
func FormatStatusWarning(msg string) string {
	return theme.FormatWarning(msg)
}

// FormatStatusInfo formats an info message for the status bar.
func FormatStatusInfo(msg string) string {
	return theme.FormatInfo(msg)
}

func isPaneClosedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.HasPrefix(msg, "pane closed") {
		return true
	}
	if strings.Contains(msg, "pane") && strings.Contains(msg, "not found") {
		return true
	}
	return false
}

func paneClosedMessage(err error) string {
	if err == nil {
		return "Pane closed"
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "Pane closed"
	}
	lower := strings.ToLower(msg)
	if strings.HasPrefix(lower, "pane closed") {
		return capitalizeFirst(msg)
	}
	if strings.Contains(lower, "pane") && strings.Contains(lower, "not found") {
		return "Pane closed"
	}
	return "Pane closed"
}

func newPaneClosedMsg(paneID string, err error) PaneClosedMsg {
	return PaneClosedMsg{
		PaneID:  paneID,
		Message: paneClosedMessage(err),
	}
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
