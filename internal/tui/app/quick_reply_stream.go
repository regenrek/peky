package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	quickReplyStreamFlushDelay   = 15 * time.Millisecond
	quickReplyStreamMaxBufBytes  = 64 * 1024
	quickReplyStreamToastMinTick = 2 * time.Second
)

type quickReplyStreamFlushMsg struct {
	Gen    uint64
	PaneID string
}

func (m *Model) quickReplyStreamEnabled() bool {
	if m == nil {
		return false
	}
	if m.state != StateDashboard || m.filterActive || m.terminalFocus {
		return false
	}
	if m.quickReplyMode != quickReplyModePane {
		return false
	}
	return m.pekyConfig().QuickReply.StreamToPane
}

func (m *Model) dropQuickReplyStreamBuffer(reason string) {
	if m == nil {
		return
	}
	if len(m.quickReplyStreamBuf) == 0 && !m.quickReplyStreamFlush {
		return
	}
	m.quickReplyStreamBuf = nil
	m.quickReplyStreamFlush = false
	m.quickReplyStreamGen++
	if strings.TrimSpace(reason) == "" {
		return
	}
	now := time.Now()
	if !m.quickReplyStreamToast.IsZero() && now.Sub(m.quickReplyStreamToast) < quickReplyStreamToastMinTick {
		return
	}
	m.quickReplyStreamToast = now
	m.setToast(reason, toastInfo)
}

func (m *Model) handleQuickReplyStreamFlush(msg quickReplyStreamFlushMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if msg.Gen == 0 || msg.Gen != m.quickReplyStreamGen {
		return nil
	}
	if msg.PaneID == "" || msg.PaneID != m.quickReplyStreamPaneID {
		return nil
	}
	m.quickReplyStreamFlush = false
	if len(m.quickReplyStreamBuf) == 0 {
		return nil
	}
	payload := append([]byte(nil), m.quickReplyStreamBuf...)
	m.quickReplyStreamBuf = nil
	return m.sendPaneInputToIDCmd(msg.PaneID, payload, "quick reply stream")
}

func (m *Model) maybeQueueQuickReplyStream(msg tea.KeyMsg) tea.Cmd {
	if !m.quickReplyStreamEnabled() {
		m.dropQuickReplyStreamBuffer("")
		return nil
	}
	if m.quickReplyHistoryActive() {
		return nil
	}

	value := m.quickReplyInput.Value()
	trimLeft := strings.TrimLeft(value, " \t")
	if strings.HasPrefix(trimLeft, "/") {
		return nil
	}
	if msg.Type == tea.KeyRunes {
		if trimLeft == "" && string(msg.Runes) == "/" {
			return nil
		}
	} else if msg.String() == "backspace" || msg.String() == "delete" {
		if value == "" {
			return nil
		}
	} else if msg.String() == "enter" {
		return nil
	} else {
		return nil
	}

	payload := encodeKeyMsg(msg)
	if len(payload) == 0 {
		return nil
	}
	pane := m.selectedPane()
	if pane == nil || strings.TrimSpace(pane.ID) == "" {
		return nil
	}
	paneID := strings.TrimSpace(pane.ID)
	if m.isPaneInputDisabled(paneID) || pane.Dead || pane.Disconnected {
		m.dropQuickReplyStreamBuffer("Stream paused: pane unavailable")
		return nil
	}
	if m.quickReplyStreamPaneID != "" && m.quickReplyStreamPaneID != paneID {
		m.dropQuickReplyStreamBuffer("Stream dropped: pane changed")
	}
	m.quickReplyStreamPaneID = paneID

	if len(m.quickReplyStreamBuf)+len(payload) > quickReplyStreamMaxBufBytes {
		m.quickReplyStreamBuf = append(m.quickReplyStreamBuf, payload...)
		gen := m.quickReplyStreamGen + 1
		m.quickReplyStreamGen = gen
		m.quickReplyStreamFlush = false
		out := quickReplyStreamFlushMsg{Gen: gen, PaneID: paneID}
		return func() tea.Msg { return out }
	}

	m.quickReplyStreamBuf = append(m.quickReplyStreamBuf, payload...)
	if m.quickReplyStreamFlush {
		return nil
	}
	gen := m.quickReplyStreamGen + 1
	m.quickReplyStreamGen = gen
	m.quickReplyStreamFlush = true
	return tea.Tick(quickReplyStreamFlushDelay, func(time.Time) tea.Msg {
		return quickReplyStreamFlushMsg{Gen: gen, PaneID: paneID}
	})
}

func (m *Model) flushQuickReplyStreamWithEnter(paneID string) tea.Cmd {
	if m == nil || strings.TrimSpace(paneID) == "" {
		return nil
	}
	if len(m.quickReplyStreamBuf) == 0 {
		return m.sendPaneInputToIDCmd(paneID, []byte{'\r'}, "quick reply stream")
	}
	payload := append([]byte(nil), m.quickReplyStreamBuf...)
	payload = append(payload, '\r')
	m.quickReplyStreamBuf = nil
	m.quickReplyStreamFlush = false
	m.quickReplyStreamGen++
	return m.sendPaneInputToIDCmd(paneID, payload, "quick reply stream")
}
