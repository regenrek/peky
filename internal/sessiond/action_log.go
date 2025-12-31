package sessiond

import (
	"sort"
	"time"
)

const defaultActionLogCap = 200

type actionLog struct {
	max   int
	start int
	count int
	buf   []PaneHistoryEntry
}

func newActionLog(max int) *actionLog {
	if max <= 0 {
		max = defaultActionLogCap
	}
	return &actionLog{
		max: max,
		buf: make([]PaneHistoryEntry, max),
	}
}

func (l *actionLog) add(entry PaneHistoryEntry) {
	if l == nil || l.max <= 0 {
		return
	}
	if l.count < l.max {
		idx := (l.start + l.count) % l.max
		l.buf[idx] = entry
		l.count++
		return
	}
	l.buf[l.start] = entry
	l.start = (l.start + 1) % l.max
}

func (l *actionLog) list(limit int, since time.Time) []PaneHistoryEntry {
	if l == nil || l.count == 0 {
		return nil
	}
	if limit <= 0 || limit > l.count {
		limit = l.count
	}
	out := make([]PaneHistoryEntry, 0, limit)
	for i := 0; i < l.count; i++ {
		entry := l.buf[(l.start+i)%l.max]
		if !since.IsZero() && entry.TS.Before(since) {
			continue
		}
		out = append(out, entry)
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TS.Before(out[j].TS)
	})
	return out
}
