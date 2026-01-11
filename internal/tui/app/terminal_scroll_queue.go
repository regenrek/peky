package app

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

const (
	terminalScrollQueueMax          = 64
	terminalScrollLinesPerRPCMax    = 8192
	terminalScrollWheelFlushDelay   = 16 * time.Millisecond
	terminalScrollDroppedToastHint  = "Scroll events dropped (scroll too fast)"
	terminalScrollDirectionDisabled = sessiond.TerminalActionUnknown
)

type queuedTerminalScroll struct {
	paneID string
	action sessiond.TerminalAction
	lines  int
}

type terminalScrollPumpResultMsg struct {
	seq  uint64
	sent int
	err  error
}

type terminalScrollWheelFlushMsg struct {
	seq uint64
}

func (m *Model) enqueueTerminalScroll(paneID string, action sessiond.TerminalAction, lines int) tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil
	}
	if lines <= 0 {
		return nil
	}
	if action == terminalScrollDirectionDisabled {
		return nil
	}

	m.enqueueTerminalScrollEvent(paneID, action, lines)
	if m.terminalScrollInFlight {
		return nil
	}
	if len(m.terminalScrollQueue) == 0 {
		return nil
	}
	return m.scheduleTerminalScrollFlush(terminalScrollWheelFlushDelay)
}

func (m *Model) enqueueTerminalScrollEvent(paneID string, action sessiond.TerminalAction, lines int) {
	if m == nil {
		return
	}
	if len(m.terminalScrollQueue) >= terminalScrollQueueMax {
		keep := terminalScrollQueueMax / 2
		if keep < 1 {
			keep = 1
		}
		m.terminalScrollQueue = append([]queuedTerminalScroll(nil), m.terminalScrollQueue[len(m.terminalScrollQueue)-keep:]...)
		m.setToast(terminalScrollDroppedToastHint, toastInfo)
	}

	if last := m.lastTerminalScrollQueueItem(); last != nil {
		if strings.TrimSpace(last.paneID) == paneID && last.action == action {
			last.lines = minInt(terminalScrollLinesPerRPCMax*4, last.lines+lines)
			return
		}
	}
	m.terminalScrollQueue = append(m.terminalScrollQueue, queuedTerminalScroll{
		paneID: paneID,
		action: action,
		lines:  lines,
	})
}

func (m *Model) lastTerminalScrollQueueItem() *queuedTerminalScroll {
	if m == nil || len(m.terminalScrollQueue) == 0 {
		return nil
	}
	return &m.terminalScrollQueue[len(m.terminalScrollQueue)-1]
}

func (m *Model) startTerminalScrollPump() tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	if m.terminalScrollInFlight {
		return nil
	}
	if len(m.terminalScrollQueue) == 0 {
		return nil
	}

	m.terminalScrollInFlight = true
	m.terminalScrollSeq++
	seq := m.terminalScrollSeq

	item := m.terminalScrollQueue[0]
	lines := minInt(item.lines, terminalScrollLinesPerRPCMax)
	if lines < 1 {
		lines = 1
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		_, err := m.client.TerminalAction(ctx, sessiond.TerminalActionRequest{
			PaneID: item.paneID,
			Action: item.action,
			Lines:  lines,
		})
		if err != nil {
			return terminalScrollPumpResultMsg{seq: seq, sent: 0, err: err}
		}
		return terminalScrollPumpResultMsg{seq: seq, sent: lines}
	}
}

func (m *Model) handleTerminalScrollPumpResult(msg terminalScrollPumpResultMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if msg.seq != m.terminalScrollSeq {
		return nil
	}
	m.terminalScrollInFlight = false

	if len(m.terminalScrollQueue) == 0 {
		return nil
	}

	head := m.terminalScrollQueue[0]
	if msg.sent > 0 {
		head.lines -= msg.sent
		if head.lines <= 0 {
			m.terminalScrollQueue = m.terminalScrollQueue[1:]
		} else {
			m.terminalScrollQueue[0] = head
		}
	}

	if msg.err != nil {
		m.terminalScrollQueue = nil
		m.setToast(ErrorMsg{Err: msg.err, Context: "terminal scroll"}.Error(), toastError)
		return nil
	}
	if len(m.terminalScrollQueue) == 0 {
		return nil
	}
	// Don't aggressively drain; rate-limit to a flush tick to keep the UI responsive
	// under trackpad momentum.
	return m.scheduleTerminalScrollFlush(terminalScrollWheelFlushDelay)
}

func (m *Model) handleTerminalScrollWheelFlush(msg terminalScrollWheelFlushMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if msg.seq != m.terminalScrollWheelFlushSeq {
		return nil
	}
	m.terminalScrollWheelFlushActive = false
	if m.terminalScrollInFlight {
		return nil
	}
	return m.startTerminalScrollPump()
}

func (m *Model) scheduleTerminalScrollFlush(delay time.Duration) tea.Cmd {
	if m == nil {
		return nil
	}
	if m.terminalScrollWheelFlushActive {
		return nil
	}
	if delay < 0 {
		delay = 0
	}
	m.terminalScrollWheelFlushActive = true
	m.terminalScrollWheelFlushSeq++
	seq := m.terminalScrollWheelFlushSeq
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return terminalScrollWheelFlushMsg{seq: seq}
	})
}

func (m *Model) terminalScrollBusy() bool {
	if m == nil {
		return false
	}
	if m.terminalScrollInFlight || m.terminalScrollWheelFlushActive {
		return true
	}
	return len(m.terminalScrollQueue) > 0
}
