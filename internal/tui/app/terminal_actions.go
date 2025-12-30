package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/termkeys"
)

const terminalActionTimeout = 2 * time.Second

func (m *Model) handleTerminalKeyCmd(msg tea.KeyMsg) tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	pane := m.selectedPane()
	if pane == nil || strings.TrimSpace(pane.ID) == "" {
		return nil
	}
	scrollToggle := key.Matches(msg, m.keys.scrollback)
	copyToggle := key.Matches(msg, m.keys.copyMode)
	keyStr := msg.String()
	if !shouldHandleTerminalKey(keyStr, scrollToggle, copyToggle) {
		return nil
	}
	payload := encodeKeyMsg(msg)
	paneID := pane.ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		resp, err := m.client.HandleTerminalKey(ctx, sessiond.TerminalKeyRequest{
			PaneID:           paneID,
			Key:              keyStr,
			ScrollbackToggle: scrollToggle,
			CopyToggle:       copyToggle,
		})
		if err != nil {
			if isPaneClosedError(err) {
				return newPaneClosedMsg(paneID, err)
			}
			return ErrorMsg{Err: err, Context: "terminal key"}
		}
		return m.handleTerminalKeyResponse(ctx, paneID, payload, resp)
	}
}

func (m *Model) sendPaneInputCmd(payload []byte, contextLabel string) tea.Cmd {
	if m == nil || m.client == nil {
		return NewErrorCmd(errors.New("session client unavailable"), contextLabel)
	}
	pane := m.selectedPane()
	if pane == nil || strings.TrimSpace(pane.ID) == "" {
		return NewWarningCmd("No pane selected")
	}
	paneID := pane.ID
	if m.isPaneInputDisabled(paneID) {
		return nil
	}
	if pane.Dead {
		return func() tea.Msg {
			return newPaneClosedMsg(paneID, nil)
		}
	}
	return func() tea.Msg {
		if m.isPaneInputDisabled(paneID) {
			return nil
		}
		if pane := m.paneByID(paneID); pane != nil && pane.Dead {
			return newPaneClosedMsg(paneID, nil)
		}
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if err := m.client.SendInput(ctx, paneID, payload); err != nil {
			if isPaneClosedError(err) {
				return newPaneClosedMsg(paneID, err)
			}
			return ErrorMsg{Err: err, Context: contextLabel}
		}
		return nil
	}
}

func isTerminalControlKey(key string) bool {
	if termkeys.IsCopyShortcutKey(key) {
		return true
	}
	switch key {
	case "esc", "q", "v", "y", "g", "G":
		return true
	case "up", "down", "left", "right", "pgup", "pgdown", "home", "end":
		return true
	case "h", "j", "k", "l":
		return true
	default:
		return false
	}
}

func shouldHandleTerminalKey(key string, scrollToggle, copyToggle bool) bool {
	return scrollToggle || copyToggle || isTerminalControlKey(key)
}

func (m *Model) handleTerminalKeyResponse(ctx context.Context, paneID string, payload []byte, resp sessiond.TerminalKeyResponse) tea.Msg {
	if resp.Handled {
		return toastFromTerminalResponse(resp)
	}
	if len(payload) == 0 {
		return nil
	}
	if m == nil || m.client == nil {
		return ErrorMsg{Err: errors.New("session client unavailable"), Context: "send to pane"}
	}
	if m.isPaneInputDisabled(paneID) {
		return nil
	}
	if pane := m.paneByID(paneID); pane == nil || pane.Dead {
		return newPaneClosedMsg(paneID, nil)
	}
	if err := m.client.SendInput(ctx, paneID, payload); err != nil {
		if isPaneClosedError(err) {
			return newPaneClosedMsg(paneID, err)
		}
		return ErrorMsg{Err: err, Context: "send to pane"}
	}
	return nil
}

func toastFromTerminalResponse(resp sessiond.TerminalKeyResponse) tea.Msg {
	if resp.YankText != "" {
		if err := clipboard.WriteAll(resp.YankText); err != nil {
			return ErrorMsg{Err: err, Context: "copy to clipboard"}
		}
	}
	if resp.Toast == "" {
		return nil
	}
	switch resp.ToastKind {
	case sessiond.ToastSuccess:
		return SuccessMsg{Message: resp.Toast}
	case sessiond.ToastWarning:
		return WarningMsg{Message: resp.Toast}
	default:
		return InfoMsg{Message: resp.Toast}
	}
}
