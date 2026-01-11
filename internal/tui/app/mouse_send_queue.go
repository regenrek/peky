package app

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

const (
	mouseSendQueueMax      = 128
	mouseSendWheelCountMax = 2048
	// Upper bound of wheel ticks per single RPC; keep reasonably large so trackpad
	// momentum collapses to a small number of RPCs.
	mouseSendWheelBurstMax  = 256
	mouseSendMotionDropHint = "Mouse events dropped (scroll too fast)"

	mouseSendWheelFlushDelay    = 16 * time.Millisecond
	mouseSendWheelIdleThreshold = 100 * time.Millisecond
)

type queuedMouseEvent struct {
	paneID  string
	payload sessiond.MouseEventPayload
	count   int // wheel only; otherwise 1
}

type mouseSendPumpResultMsg struct {
	seq  uint64
	sent int
	err  error
}

type mouseSendWheelFlushMsg struct {
	seq uint64
}

func (m *Model) enqueueMouseSend(paneID string, payload sessiond.MouseEventPayload) tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return nil
	}

	m.enqueueMouseEvent(paneID, payload)
	if m.mouseSendInFlight {
		return nil
	}
	if len(m.mouseSendQueue) == 0 {
		return nil
	}
	if m.mouseSendQueue[0].payload.Wheel {
		return m.maybeStartWheelSend()
	}
	return m.startMouseSendPump()
}

func (m *Model) enqueueMouseEvent(paneID string, payload sessiond.MouseEventPayload) {
	if m == nil {
		return
	}
	if len(m.mouseSendQueue) >= mouseSendQueueMax {
		keep := mouseSendQueueMax / 2
		if keep < 1 {
			keep = 1
		}
		m.mouseSendQueue = append([]queuedMouseEvent(nil), m.mouseSendQueue[len(m.mouseSendQueue)-keep:]...)
		m.setToast(mouseSendMotionDropHint, toastInfo)
	}

	if payload.Wheel {
		if last := m.lastMouseQueueItem(); last != nil && canCoalesceWheel(*last, paneID, payload) {
			last.count = minInt(mouseSendWheelCountMax, last.count+1)
			last.payload = payload
			return
		}
		m.mouseSendQueue = append(m.mouseSendQueue, queuedMouseEvent{paneID: paneID, payload: payload, count: 1})
		return
	}

	if payload.Action == sessiond.MouseActionMotion {
		if last := m.lastMouseQueueItem(); last != nil && canCoalesceMotion(*last, paneID, payload) {
			last.payload = payload
			return
		}
	}

	m.mouseSendQueue = append(m.mouseSendQueue, queuedMouseEvent{paneID: paneID, payload: payload, count: 1})
}

func (m *Model) lastMouseQueueItem() *queuedMouseEvent {
	if m == nil || len(m.mouseSendQueue) == 0 {
		return nil
	}
	return &m.mouseSendQueue[len(m.mouseSendQueue)-1]
}

func canCoalesceWheel(last queuedMouseEvent, paneID string, payload sessiond.MouseEventPayload) bool {
	if !last.payload.Wheel {
		return false
	}
	if strings.TrimSpace(last.paneID) != strings.TrimSpace(paneID) {
		return false
	}
	// Keep wheel direction + modifiers stable; coordinates can drift.
	if last.payload.Button != payload.Button {
		return false
	}
	if last.payload.Shift != payload.Shift || last.payload.Alt != payload.Alt || last.payload.Ctrl != payload.Ctrl {
		return false
	}
	if last.payload.Route != payload.Route {
		return false
	}
	return true
}

func canCoalesceMotion(last queuedMouseEvent, paneID string, payload sessiond.MouseEventPayload) bool {
	if last.payload.Action != sessiond.MouseActionMotion {
		return false
	}
	if strings.TrimSpace(last.paneID) != strings.TrimSpace(paneID) {
		return false
	}
	if last.payload.Button != payload.Button {
		return false
	}
	if last.payload.Shift != payload.Shift || last.payload.Alt != payload.Alt || last.payload.Ctrl != payload.Ctrl {
		return false
	}
	if last.payload.Route != payload.Route {
		return false
	}
	return true
}

func (m *Model) startMouseSendPump() tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	if m.mouseSendInFlight {
		return nil
	}
	if len(m.mouseSendQueue) == 0 {
		return nil
	}

	m.mouseSendInFlight = true
	m.mouseSendSeq++
	seq := m.mouseSendSeq

	item := m.mouseSendQueue[0]
	sendCount := 1
	payload := item.payload
	if payload.Wheel {
		sendCount = minInt(item.count, mouseSendWheelBurstMax)
		if sendCount < 1 {
			sendCount = 1
		}
		payload.WheelCount = sendCount
	} else {
		payload.WheelCount = 0
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if err := m.client.SendMouse(ctx, item.paneID, payload); err != nil {
			return mouseSendPumpResultMsg{seq: seq, sent: 0, err: err}
		}
		return mouseSendPumpResultMsg{seq: seq, sent: sendCount}
	}
}

func (m *Model) maybeStartWheelSend() tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	now := time.Now()
	idle := m.mouseSendWheelLastAt.IsZero() || now.Sub(m.mouseSendWheelLastAt) >= mouseSendWheelIdleThreshold
	m.mouseSendWheelLastAt = now

	if idle {
		m.mouseSendWheelFlushScheduled = false
		return m.startMouseSendPump()
	}

	if m.mouseSendWheelFlushScheduled {
		return nil
	}
	m.mouseSendWheelFlushScheduled = true
	m.mouseSendWheelFlushSeq++
	seq := m.mouseSendWheelFlushSeq
	return tea.Tick(mouseSendWheelFlushDelay, func(time.Time) tea.Msg {
		return mouseSendWheelFlushMsg{seq: seq}
	})
}

func (m *Model) handleMouseSendPumpResult(msg mouseSendPumpResultMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if msg.seq != m.mouseSendSeq {
		return nil
	}
	m.mouseSendInFlight = false

	if len(m.mouseSendQueue) == 0 {
		return nil
	}

	head := m.mouseSendQueue[0]
	if head.payload.Wheel && head.count > 0 {
		head.count -= msg.sent
		if head.count <= 0 {
			m.mouseSendQueue = m.mouseSendQueue[1:]
		} else {
			m.mouseSendQueue[0] = head
		}
	} else if msg.sent > 0 {
		m.mouseSendQueue = m.mouseSendQueue[1:]
	}

	if msg.err != nil {
		m.mouseSendQueue = nil
		m.setToast(ErrorMsg{Err: msg.err, Context: "send mouse"}.Error(), toastError)
		return nil
	}
	return m.startMouseSendPump()
}

func (m *Model) handleMouseSendWheelFlush(msg mouseSendWheelFlushMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if msg.seq != m.mouseSendWheelFlushSeq {
		return nil
	}
	m.mouseSendWheelFlushScheduled = false
	if m.mouseSendInFlight {
		return nil
	}
	return m.startMouseSendPump()
}
