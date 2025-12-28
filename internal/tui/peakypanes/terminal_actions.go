package peakypanes

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
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
	if !scrollToggle && !copyToggle && !isTerminalControlKey(keyStr) {
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
			return ErrorMsg{Err: err, Context: "terminal key"}
		}
		if resp.Handled {
			if resp.YankText != "" {
				if err := clipboard.WriteAll(resp.YankText); err != nil {
					return ErrorMsg{Err: err, Context: "copy to clipboard"}
				}
			}
			if resp.Toast != "" {
				switch resp.ToastKind {
				case sessiond.ToastSuccess:
					return SuccessMsg{Message: resp.Toast}
				case sessiond.ToastWarning:
					return WarningMsg{Message: resp.Toast}
				default:
					return InfoMsg{Message: resp.Toast}
				}
			}
			return nil
		}
		if len(payload) == 0 {
			return nil
		}
		if err := m.client.SendInput(ctx, paneID, payload); err != nil {
			return ErrorMsg{Err: err, Context: "send to pane"}
		}
		return nil
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
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if err := m.client.SendInput(ctx, paneID, payload); err != nil {
			return ErrorMsg{Err: err, Context: contextLabel}
		}
		return nil
	}
}

func isTerminalControlKey(key string) bool {
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
