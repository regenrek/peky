package dashboard

import (
	"io"
	"time"

	"github.com/muesli/cancelreader"
)

const escByte = 0x1b

const (
	maxMouseSeqLen       = 64
	maxPrefixWait        = 150 * time.Millisecond
	maxMouseFragmentAge  = 200 * time.Millisecond
	singlePrefixWait     = 50 * time.Millisecond
	mousePrefixWait      = 5 * time.Millisecond
	maxOutputBufferBytes = 32 * 1024
)

type inputReadyFn func(fd uintptr, timeout time.Duration) (bool, error)

type pendingWaitAction uint8

const (
	pendingWaitNone pendingWaitAction = iota
	pendingWaitContinue
	pendingWaitBreak
)

// repairedTUIInput ensures Bubble Tea never receives a chunk that ends mid escape
// sequence (especially SGR mouse) which can cause literal junk to leak into text
// inputs under heavy mouse traffic.
type repairedTUIInput struct {
	f cancelreader.File

	readyFn inputReadyFn

	pending []byte
	out     []byte
	tmp     []byte
	readErr error

	lastMouseSeqAt time.Time
}

func newRepairedTUIInput(f cancelreader.File) *repairedTUIInput {
	if f == nil {
		return &repairedTUIInput{}
	}
	return &repairedTUIInput{
		f:       f,
		readyFn: inputReady,
		tmp:     make([]byte, 32*1024),
	}
}

func (r *repairedTUIInput) Read(p []byte) (int, error) {
	if r == nil || r.f == nil {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}

	for len(r.out) == 0 {
		if r.readErr != nil {
			if r.flushPendingAfterReadErr() {
				break
			}
			return 0, r.readErr
		}

		if len(r.pending) > 0 {
			switch r.maybeHandlePendingWait() {
			case pendingWaitBreak:
				break
			case pendingWaitContinue:
				continue
			}
		}

		n, err := r.f.Read(r.tmp)
		if n > 0 {
			r.pending = append(r.pending, r.tmp[:n]...)
			r.processPending()
		}
		if err != nil {
			r.readErr = err
		}
	}

	return r.drainOut(p)
}

func (r *repairedTUIInput) drainOut(p []byte) (int, error) {
	n := safeReadLen(r.out, len(p))
	if n > 0 {
		copy(p, r.out[:n])
		r.out = r.out[n:]
		return n, nil
	}
	return 0, r.readErr
}

func (r *repairedTUIInput) flushPendingAfterReadErr() bool {
	if len(r.pending) == 0 {
		return false
	}
	if r.shouldFlushPendingAfterReadErr(r.pending) {
		r.out = append(r.out, r.pending...)
	}
	r.pending = nil
	return len(r.out) > 0
}

func (r *repairedTUIInput) shouldFlushPendingAfterReadErr(pending []byte) bool {
	if isIncompleteEscapeSequence(pending) {
		return false
	}
	if isIncompleteSGRMousePrefix(pending) {
		return false
	}
	if r.looksLikeRecentMouseFragment(pending) {
		return false
	}
	return true
}

func (r *repairedTUIInput) pendingWaitDuration() time.Duration {
	wait := maxPrefixWait
	if len(r.pending) == 1 && r.pending[0] == escByte {
		wait = singlePrefixWait
	}
	if looksLikeMousePrefix(r.pending) {
		// Mouse bursts (and fragments) must be low-latency; long waits here feel like
		// "parallax" scrolling because Bubble Tea receives wheel events late.
		wait = mousePrefixWait
	}
	return wait
}

func (r *repairedTUIInput) maybeHandlePendingWait() pendingWaitAction {
	readyFn := r.readyFn
	if readyFn == nil {
		readyFn = inputReady
	}
	ready, err := readyFn(r.f.Fd(), r.pendingWaitDuration())
	if err == nil && ready {
		return pendingWaitNone
	}
	return r.onPendingTimeout()
}

func (r *repairedTUIInput) onPendingTimeout() pendingWaitAction {
	pending := r.pending
	if len(pending) == 1 && pending[0] == escByte {
		r.out = append(r.out, escByte)
		r.pending = nil
		return pendingWaitBreak
	}
	if isIncompleteEscapeSequence(pending) || isIncompleteSGRMousePrefix(pending) || r.looksLikeRecentMouseFragment(pending) {
		r.pending = nil
		return pendingWaitContinue
	}
	r.out = append(r.out, pending...)
	r.pending = nil
	return pendingWaitBreak
}

func (r *repairedTUIInput) looksLikeRecentMouseFragment(pending []byte) bool {
	return looksLikeMousePrefix(pending) &&
		!r.lastMouseSeqAt.IsZero() &&
		time.Since(r.lastMouseSeqAt) < maxMouseFragmentAge
}

func (r *repairedTUIInput) Write(p []byte) (int, error) {
	if r == nil || r.f == nil {
		return 0, io.ErrClosedPipe
	}
	return r.f.Write(p)
}

func (r *repairedTUIInput) Close() error {
	// Bubble Tea doesn't close custom inputs; closure is managed by the caller.
	return nil
}

func (r *repairedTUIInput) Fd() uintptr {
	if r == nil || r.f == nil {
		return 0
	}
	return r.f.Fd()
}

func (r *repairedTUIInput) Name() string {
	if r == nil || r.f == nil {
		return ""
	}
	return r.f.Name()
}

func (r *repairedTUIInput) processPending() {
	b := r.pending
	if len(b) == 0 {
		return
	}

	out := r.out
	if len(out) > maxOutputBufferBytes {
		// Avoid unbounded growth; Bubble Tea will drain `out` via Read().
		out = out[:0]
	}

	i := 0
	for i < len(b) {
		next, out2, consumed, needMore := r.consumeEscaped(b, i, out)
		if consumed {
			out = out2
			if needMore {
				break
			}
			i = next
			continue
		}

		next, out2, consumed, needMore = r.consumeBracketBurstSGRMouse(b, i, out)
		if consumed {
			out = out2
			if needMore {
				break
			}
			i = next
			continue
		}

		next, out2, consumed, needMore = r.consumeBareSGRMouse(b, i, out)
		if consumed {
			out = out2
			if needMore {
				break
			}
			i = next
			continue
		}

		next, out2, consumed, needMore = r.consumeBrokenMouseFragment(b, i, out)
		if consumed {
			out = out2
			if needMore {
				break
			}
			i = next
			continue
		}

		out = append(out, b[i])
		i++
	}

	r.out = out
	r.pending = b[i:]
}

func (r *repairedTUIInput) consumeEscaped(b []byte, i int, out []byte) (next int, out2 []byte, consumed bool, needMore bool) {
	if b[i] != escByte {
		return i, out, false, false
	}
	if i == len(b)-1 {
		return i, out, true, true
	}

	next, out2, consumed, needMore = r.consumeEscSGRMouse(b, i, out)
	if consumed {
		return next, out2, true, needMore
	}

	end, ok := scanEscapeSequence(b, i)
	if !ok {
		return i, out, true, true
	}
	out = append(out, b[i:end]...)
	return end, out, true, false
}

func (r *repairedTUIInput) consumeEscSGRMouse(b []byte, i int, out []byte) (next int, out2 []byte, consumed bool, needMore bool) {
	if i+1 >= len(b) || b[i+1] != '[' {
		return i, out, false, false
	}

	k := i + 1
	for k < len(b) && b[k] == '[' {
		k++
	}
	if k >= len(b) {
		return i, out, true, true
	}
	if b[k] != '<' {
		return i, out, false, false
	}

	end, needMore, ok := scanSGRMouse(b, k)
	if needMore {
		return i, out, true, true
	}
	if !ok {
		return i, out, false, false
	}

	out = append(out, escByte, '[')
	out = append(out, b[k:end]...)
	r.lastMouseSeqAt = time.Now()
	return end, out, true, false
}

func (r *repairedTUIInput) consumeBracketBurstSGRMouse(b []byte, i int, out []byte) (next int, out2 []byte, consumed bool, needMore bool) {
	if b[i] != '[' {
		return i, out, false, false
	}

	j := i
	for j < len(b) && b[j] == '[' {
		j++
	}
	if j >= len(b) {
		return i, out, true, true
	}
	if b[j] != '<' {
		return i, out, false, false
	}

	end, needMore, ok := scanSGRMouse(b, j)
	if needMore {
		return i, out, true, true
	}
	if !ok {
		return i, out, false, false
	}

	out = append(out, escByte, '[')
	out = append(out, b[j:end]...)
	r.lastMouseSeqAt = time.Now()
	return end, out, true, false
}

func (r *repairedTUIInput) consumeBareSGRMouse(b []byte, i int, out []byte) (next int, out2 []byte, consumed bool, needMore bool) {
	if b[i] != '<' {
		return i, out, false, false
	}

	end, needMore, ok := scanSGRMouse(b, i)
	if needMore {
		if len(b)-i > maxMouseSeqLen {
			out = append(out, b[i])
			return i + 1, out, true, false
		}
		return i, out, true, true
	}
	if !ok {
		return i, out, false, false
	}

	out = append(out, escByte, '[')
	out = append(out, b[i:end]...)
	r.lastMouseSeqAt = time.Now()
	return end, out, true, false
}

func (r *repairedTUIInput) consumeBrokenMouseFragment(b []byte, i int, out []byte) (next int, out2 []byte, consumed bool, needMore bool) {
	if b[i] != ';' && (b[i] < '0' || b[i] > '9') {
		return i, out, false, false
	}

	end, needMore, ok, kind := scanSGRMouseFragment(b, i)
	if needMore {
		if len(b)-i > maxMouseSeqLen {
			out = append(out, b[i])
			return i + 1, out, true, false
		}
		return i, out, true, true
	}
	if ok && shouldDropMouseFragment(r.lastMouseSeqAt, b, end, kind) {
		return end, out, true, false
	}
	return i, out, false, false
}
