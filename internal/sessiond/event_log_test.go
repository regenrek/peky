package sessiond

import (
	"testing"
	"time"
)

func TestEventLogAddAndList(t *testing.T) {
	log := newEventLog(2)
	if events := log.list(time.Time{}, time.Time{}, 0, nil); len(events) != 0 {
		t.Fatalf("expected empty log")
	}
	log.add(Event{TS: time.Now().Add(-2 * time.Second), Type: EventPaneUpdated})
	log.add(Event{TS: time.Now().Add(-1 * time.Second), Type: EventToast})
	log.add(Event{TS: time.Now(), Type: EventFocus})
	out := log.list(time.Time{}, time.Time{}, 0, nil)
	if len(out) != 2 {
		t.Fatalf("expected 2 events, got %d", len(out))
	}
	if out[0].Type != EventToast || out[1].Type != EventFocus {
		t.Fatalf("unexpected events: %+v", out)
	}
}

func TestEventLogFilters(t *testing.T) {
	log := newEventLog(10)
	base := time.Now().Add(-10 * time.Second)
	log.add(Event{TS: base.Add(1 * time.Second), Type: EventPaneUpdated})
	log.add(Event{TS: base.Add(2 * time.Second), Type: EventToast})
	log.add(Event{TS: base.Add(3 * time.Second), Type: EventFocus})

	types := map[EventType]struct{}{EventToast: {}}
	out := log.list(base.Add(1500*time.Millisecond), base.Add(2500*time.Millisecond), 10, types)
	if len(out) != 1 {
		t.Fatalf("expected 1 event, got %d", len(out))
	}
	if out[0].Type != EventToast {
		t.Fatalf("unexpected event: %+v", out[0])
	}
}

func TestRecordEvent(t *testing.T) {
	d := &Daemon{eventLog: newEventLog(10)}
	event := d.recordEvent(Event{Type: EventRelay})
	if event.ID == "" {
		t.Fatalf("expected event id")
	}
	if event.TS.IsZero() {
		t.Fatalf("expected timestamp")
	}
	out := d.eventLog.list(time.Time{}, time.Time{}, 10, nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 event, got %d", len(out))
	}
}
