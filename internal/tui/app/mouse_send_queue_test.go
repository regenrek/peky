package app

import (
	"testing"

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
