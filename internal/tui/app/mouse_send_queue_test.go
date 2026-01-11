package app

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestMouseSendQueueCoalescesWheelCounts(t *testing.T) {
	m := &Model{}
	payload := sessiond.MouseEventPayload{
		X:      1,
		Y:      2,
		Button: 4,
		Wheel:  true,
		Route:  sessiond.MouseRouteAuto,
	}
	for i := 0; i < 10; i++ {
		m.enqueueMouseEvent("p-1", payload)
	}
	if len(m.mouseSendQueue) != 1 {
		t.Fatalf("queue len=%d", len(m.mouseSendQueue))
	}
	if got := m.mouseSendQueue[0].count; got != 10 {
		t.Fatalf("count=%d", got)
	}
}

func TestMouseSendQueueDoesNotCoalesceWheelDirectionChange(t *testing.T) {
	m := &Model{}
	up := sessiond.MouseEventPayload{Button: 4, Wheel: true, Route: sessiond.MouseRouteAuto}
	down := sessiond.MouseEventPayload{Button: 5, Wheel: true, Route: sessiond.MouseRouteAuto}
	m.enqueueMouseEvent("p-1", up)
	m.enqueueMouseEvent("p-1", down)
	if len(m.mouseSendQueue) != 2 {
		t.Fatalf("queue len=%d", len(m.mouseSendQueue))
	}
}

func TestMouseSendQueueWheelCountClamped(t *testing.T) {
	m := &Model{}
	payload := sessiond.MouseEventPayload{Button: 4, Wheel: true, Route: sessiond.MouseRouteAuto}
	for i := 0; i < mouseSendWheelCountMax+100; i++ {
		m.enqueueMouseEvent("p-1", payload)
	}
	if len(m.mouseSendQueue) != 1 {
		t.Fatalf("queue len=%d", len(m.mouseSendQueue))
	}
	if got := m.mouseSendQueue[0].count; got != mouseSendWheelCountMax {
		t.Fatalf("count=%d", got)
	}
}

func TestMouseSendWheelIdleStartsPumpImmediately(t *testing.T) {
	m := &Model{client: &sessiond.Client{}}
	payload := sessiond.MouseEventPayload{Button: 4, Wheel: true, Route: sessiond.MouseRouteAuto}
	cmd := m.enqueueMouseSend("p-1", payload)
	if cmd == nil {
		t.Fatalf("cmd=nil")
	}
	if !m.mouseSendInFlight {
		t.Fatalf("mouseSendInFlight=false")
	}
	if m.mouseSendWheelFlushScheduled {
		t.Fatalf("mouseSendWheelFlushScheduled=true")
	}
}

func TestMouseSendWheelBurstSchedulesFlush(t *testing.T) {
	m := &Model{client: &sessiond.Client{}}
	m.mouseSendWheelLastAt = time.Now()
	payload := sessiond.MouseEventPayload{Button: 4, Wheel: true, Route: sessiond.MouseRouteAuto}

	cmd := m.enqueueMouseSend("p-1", payload)
	if cmd == nil {
		t.Fatalf("cmd=nil")
	}
	if m.mouseSendInFlight {
		t.Fatalf("mouseSendInFlight=true")
	}
	if !m.mouseSendWheelFlushScheduled {
		t.Fatalf("mouseSendWheelFlushScheduled=false")
	}
	seq := m.mouseSendWheelFlushSeq

	cmd2 := m.enqueueMouseSend("p-1", payload)
	if cmd2 != nil {
		t.Fatalf("cmd2!=nil")
	}
	if m.mouseSendWheelFlushSeq != seq {
		t.Fatalf("seq=%d want=%d", m.mouseSendWheelFlushSeq, seq)
	}
}

func TestMouseSendWheelFlushClearsScheduledAndStartsPump(t *testing.T) {
	m := &Model{client: &sessiond.Client{}}
	m.mouseSendWheelFlushScheduled = true
	m.mouseSendWheelFlushSeq = 7
	m.mouseSendQueue = []queuedMouseEvent{
		{paneID: "p-1", payload: sessiond.MouseEventPayload{Button: 4, Wheel: true, Route: sessiond.MouseRouteAuto}, count: 3},
	}
	cmd := m.handleMouseSendWheelFlush(mouseSendWheelFlushMsg{seq: 7})
	if cmd == nil {
		t.Fatalf("cmd=nil")
	}
	if m.mouseSendWheelFlushScheduled {
		t.Fatalf("mouseSendWheelFlushScheduled=true")
	}
	if !m.mouseSendInFlight {
		t.Fatalf("mouseSendInFlight=false")
	}
}
