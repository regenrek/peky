package native

import (
	"bytes"
	"context"
	"sync"
	"time"
)

const (
	defaultOutputLineCap = 2000
)

// OutputLine is a single line of pane output.
type OutputLine struct {
	Seq  uint64
	TS   time.Time
	Text string
}

// OutputChunk is a raw output chunk.
type OutputChunk struct {
	TS        time.Time
	Data      []byte
	Truncated bool
}

type rawSubscriber struct {
	ch      chan OutputChunk
	dropped uint64
}

type outputLog struct {
	mu       sync.Mutex
	max      int
	seq      uint64
	buf      []OutputLine
	start    int
	count    int
	partial  []byte
	notify   chan struct{}
	rawSubs  map[uint64]*rawSubscriber
	nextSub  uint64
	disabled bool
}

func newOutputLog(max int) *outputLog {
	if max <= 0 {
		max = defaultOutputLineCap
	}
	return &outputLog{
		max:     max,
		buf:     make([]OutputLine, max),
		notify:  make(chan struct{}, 1),
		rawSubs: make(map[uint64]*rawSubscriber),
	}
}

func (o *outputLog) append(data []byte) {
	if o == nil || len(data) == 0 {
		return
	}
	o.mu.Lock()
	if o.disabled {
		o.mu.Unlock()
		return
	}
	ts := time.Now().UTC()
	o.broadcastRawLocked(ts, data)
	o.splitLinesLocked(ts, data)
	o.mu.Unlock()
}

func (o *outputLog) disable() {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.disabled = true
	for id, sub := range o.rawSubs {
		close(sub.ch)
		delete(o.rawSubs, id)
	}
	o.mu.Unlock()
}

func (o *outputLog) splitLinesLocked(ts time.Time, data []byte) {
	o.partial = append(o.partial, data...)
	for {
		idx := bytes.IndexByte(o.partial, '\n')
		if idx < 0 {
			break
		}
		line := bytes.TrimSuffix(o.partial[:idx], []byte{'\r'})
		o.appendLineLocked(ts, string(line))
		o.partial = o.partial[idx+1:]
	}
	if len(o.partial) > 1<<20 {
		line := bytes.TrimSuffix(o.partial, []byte{'\r'})
		o.appendLineLocked(ts, string(line))
		o.partial = nil
	}
}

func (o *outputLog) appendLineLocked(ts time.Time, text string) {
	o.seq++
	entry := OutputLine{Seq: o.seq, TS: ts, Text: text}
	if o.max <= 0 {
		return
	}
	if o.count < o.max {
		idx := (o.start + o.count) % o.max
		o.buf[idx] = entry
		o.count++
	} else {
		o.buf[o.start] = entry
		o.start = (o.start + 1) % o.max
	}
	select {
	case o.notify <- struct{}{}:
	default:
	}
}

func (o *outputLog) broadcastRawLocked(ts time.Time, data []byte) {
	if len(o.rawSubs) == 0 {
		return
	}
	payload := append([]byte(nil), data...)
	for _, sub := range o.rawSubs {
		trunc := sub.dropped > 0
		select {
		case sub.ch <- OutputChunk{TS: ts, Data: payload, Truncated: trunc}:
			sub.dropped = 0
		default:
			sub.dropped++
		}
	}
}

func (o *outputLog) linesSince(seq uint64) ([]OutputLine, uint64, bool) {
	if o == nil {
		return nil, seq, false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.count == 0 {
		return nil, seq, false
	}
	minSeq := o.seq - uint64(o.count) + 1
	truncated := false
	if seq < minSeq-1 {
		truncated = true
		seq = minSeq - 1
	}
	lines := make([]OutputLine, 0, o.count)
	for i := 0; i < o.count; i++ {
		entry := o.buf[(o.start+i)%o.max]
		if entry.Seq > seq {
			lines = append(lines, entry)
		}
	}
	next := seq
	if len(lines) > 0 {
		next = lines[len(lines)-1].Seq
	}
	return lines, next, truncated
}

func (o *outputLog) snapshot(limit int) []OutputLine {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.count == 0 || limit <= 0 {
		return nil
	}
	if limit > o.count {
		limit = o.count
	}
	out := make([]OutputLine, 0, limit)
	start := o.count - limit
	for i := 0; i < limit; i++ {
		entry := o.buf[(o.start+start+i)%o.max]
		out = append(out, entry)
	}
	return out
}

func (o *outputLog) subscribeRaw(buffer int) (uint64, <-chan OutputChunk, func()) {
	if o == nil {
		return 0, nil, func() {}
	}
	if buffer <= 0 {
		buffer = 64
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.nextSub++
	id := o.nextSub
	ch := make(chan OutputChunk, buffer)
	o.rawSubs[id] = &rawSubscriber{ch: ch}
	cancel := func() {
		o.mu.Lock()
		sub := o.rawSubs[id]
		delete(o.rawSubs, id)
		o.mu.Unlock()
		if sub != nil {
			close(sub.ch)
		}
	}
	return id, ch, cancel
}

func (o *outputLog) wait(ctx context.Context) bool {
	if o == nil {
		return false
	}
	select {
	case <-o.notify:
		return true
	case <-ctx.Done():
		return false
	}
}
