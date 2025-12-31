package sessiond

import (
	"sort"
	"time"
)

const defaultEventLogCap = 1000

type eventLog struct {
	max   int
	start int
	count int
	buf   []Event
}

func newEventLog(max int) *eventLog {
	if max <= 0 {
		max = defaultEventLogCap
	}
	return &eventLog{
		max: max,
		buf: make([]Event, max),
	}
}

func (l *eventLog) add(event Event) {
	if l == nil || l.max <= 0 {
		return
	}
	if l.count < l.max {
		idx := (l.start + l.count) % l.max
		l.buf[idx] = event
		l.count++
		return
	}
	l.buf[l.start] = event
	l.start = (l.start + 1) % l.max
}

func (l *eventLog) list(since, until time.Time, limit int, types map[EventType]struct{}) []Event {
	if l == nil || l.count == 0 {
		return nil
	}
	if limit <= 0 || limit > l.count {
		limit = l.count
	}
	out := make([]Event, 0, limit)
	for i := 0; i < l.count; i++ {
		event := l.buf[(l.start+i)%l.max]
		if !since.IsZero() && event.TS.Before(since) {
			continue
		}
		if !until.IsZero() && event.TS.After(until) {
			continue
		}
		if len(types) > 0 {
			if _, ok := types[event.Type]; !ok {
				continue
			}
		}
		out = append(out, event)
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TS.Before(out[j].TS)
	})
	return out
}
