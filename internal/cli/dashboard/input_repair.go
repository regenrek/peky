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
		if len(r.pending) > 0 && r.readErr == nil {
			readyFn := r.readyFn
			if readyFn == nil {
				readyFn = inputReady
			}
			wait := maxPrefixWait
			if len(r.pending) == 1 && r.pending[0] == escByte {
				wait = singlePrefixWait
			}
			if looksLikeMousePrefix(r.pending) {
				// Mouse bursts (and fragments) must be low-latency; long waits here
				// feel like "parallax" scrolling because Bubble Tea receives wheel
				// events late.
				wait = mousePrefixWait
			}
			ready, err := readyFn(r.f.Fd(), wait)
			if err != nil || !ready {
				pending := r.pending

				// If we time out while holding a likely-broken mouse prefix, drop it.
				// This prevents junk like `[[`, `<64;..`, `5;70;19M` leaking into text
				// inputs during heavy mouse traffic.
				switch {
				case len(pending) == 1 && pending[0] == escByte:
					r.out = append(r.out, escByte)
				case isIncompleteEscapeSequence(pending):
					// Never flush partial ESC-prefixed sequences as text. If they reach
					// r.out, safeReadLen() can split them into short reads (ESC then '['),
					// and Bubble Tea will treat '[' as literal input.
					r.pending = nil
					continue
				case isIncompleteSGRMousePrefix(pending):
					// Drop and keep reading; never flush incomplete mouse prefixes as text.
					r.pending = nil
					continue
				case !r.lastMouseSeqAt.IsZero() && time.Since(r.lastMouseSeqAt) < maxMouseFragmentAge && looksLikeMousePrefix(pending):
					// Drop and keep reading; don't leak stray '[' during mouse bursts.
					r.pending = nil
					continue
				default:
					r.out = append(r.out, pending...)
				}

				r.pending = nil
				break
			}
		}

		if r.readErr != nil {
			if len(r.pending) > 0 {
				// Avoid flushing broken mouse prefixes on shutdown/errors.
				if !isIncompleteEscapeSequence(r.pending) &&
					!isIncompleteSGRMousePrefix(r.pending) &&
					!(looksLikeMousePrefix(r.pending) && !r.lastMouseSeqAt.IsZero() && time.Since(r.lastMouseSeqAt) < maxMouseFragmentAge) {
					r.out = append(r.out, r.pending...)
				}
				r.pending = nil
				if len(r.out) > 0 {
					break
				}
			}
			return 0, r.readErr
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

	n := safeReadLen(r.out, len(p))
	if n > 0 {
		copy(p, r.out[:n])
		r.out = r.out[n:]
		return n, nil
	}
	return 0, r.readErr
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
		if b[i] == escByte {
			if i == len(b)-1 {
				break
			}

			// Handle SGR mouse sequences (and bracket bursts) as a unit, and
			// normalize ESC[[<... into ESC[<...
			if b[i+1] == '[' {
				k := i + 1
				for k < len(b) && b[k] == '[' {
					k++
				}
				if k >= len(b) {
					break
				}
				if b[k] == '<' {
					end, needMore, ok := scanSGRMouse(b, k)
					if needMore {
						break
					}
					if ok {
						out = append(out, escByte, '[')
						out = append(out, b[k:end]...)
						r.lastMouseSeqAt = time.Now()
						i = end
						continue
					}
				}
			}

			end, ok := scanEscapeSequence(b, i)
			if !ok {
				break
			}
			out = append(out, b[i:end]...)
			i = end
			continue
		}

		// Repair missing ESC for bracket-burst SGR sequences: "[[<...M" -> ESC[<...M.
		if b[i] == '[' {
			j := i
			for j < len(b) && b[j] == '[' {
				j++
			}
			if j >= len(b) {
				break
			}
			if b[j] == '<' {
				end, needMore, ok := scanSGRMouse(b, j)
				if needMore {
					break
				}
				if ok {
					out = append(out, escByte, '[')
					out = append(out, b[j:end]...)
					r.lastMouseSeqAt = time.Now()
					i = end
					continue
				}
			}
		}

		// Repair missing ESC[ for SGR mouse sequences: "<...M" -> ESC[<...M.
		if b[i] == '<' {
			end, needMore, ok := scanSGRMouse(b, i)
			if needMore {
				if len(b)-i > maxMouseSeqLen {
					out = append(out, b[i])
					i++
					continue
				}
				break
			}
			if ok {
				out = append(out, escByte, '[')
				out = append(out, b[i:end]...)
				r.lastMouseSeqAt = time.Now()
				i = end
				continue
			}
		}

		// Drop known-broken mouse fragments that arrive without CSI and/or "<".
		// Example seen in traces: "5;71;16M\x1b[<65;71;16M..."
		if b[i] == ';' || (b[i] >= '0' && b[i] <= '9') {
			end, needMore, ok, kind := scanSGRMouseFragment(b, i)
			if needMore {
				if len(b)-i > maxMouseSeqLen {
					needMore = false
				} else {
					break
				}
			}
			if ok && shouldDropMouseFragment(r.lastMouseSeqAt, b, end, kind) {
				i = end
				continue
			}
		}

		out = append(out, b[i])
		i++
	}

	r.out = out
	r.pending = b[i:]
}
