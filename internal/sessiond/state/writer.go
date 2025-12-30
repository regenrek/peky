package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

const defaultDebounce = 250 * time.Millisecond

// WriterOptions configures the state writer.
type WriterOptions struct {
	Debounce time.Duration
	FileMode os.FileMode
	Now      func() time.Time
}

// Writer debounces and persists runtime state to disk.
type Writer struct {
	path      string
	opts      WriterOptions
	persistCh chan RuntimeState
	flushCh   chan chan error
	closeCh   chan chan error
	closed    atomic.Bool
}

// NewWriter starts a debounced state writer.
func NewWriter(path string, opts WriterOptions) *Writer {
	if opts.Debounce < 0 {
		opts.Debounce = defaultDebounce
	}
	if opts.FileMode == 0 {
		opts.FileMode = 0o600
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	w := &Writer{
		path:      path,
		opts:      opts,
		persistCh: make(chan RuntimeState, 1),
		flushCh:   make(chan chan error),
		closeCh:   make(chan chan error),
	}
	go w.loop()
	return w
}

// Persist schedules a new state snapshot to be written.
func (w *Writer) Persist(st RuntimeState) {
	if w == nil || w.closed.Load() {
		return
	}
	select {
	case w.persistCh <- st:
		return
	default:
	}
	select {
	case <-w.persistCh:
	default:
	}
	select {
	case w.persistCh <- st:
	default:
	}
}

// Flush forces any pending state to be written.
func (w *Writer) Flush(ctx context.Context) error {
	if w == nil || w.closed.Load() {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	resp := make(chan error, 1)
	select {
	case w.flushCh <- resp:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-resp:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close flushes pending state and stops the writer.
func (w *Writer) Close(ctx context.Context) error {
	if w == nil {
		return nil
	}
	if w.closed.Swap(true) {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	resp := make(chan error, 1)
	select {
	case w.closeCh <- resp:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-resp:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Writer) loop() {
	var state writerPendingState
	for {
		select {
		case st := <-w.persistCh:
			w.handlePersist(&state, st)
		case <-state.timerCh:
			w.handleTimer(&state)
		case resp := <-w.flushCh:
			resp <- w.handleFlush(&state)
		case resp := <-w.closeCh:
			resp <- w.handleClose(&state)
			return
		}
	}
}

type writerPendingState struct {
	pending    RuntimeState
	hasPending bool
	timer      *time.Timer
	timerCh    <-chan time.Time
}

func (w *Writer) handlePersist(state *writerPendingState, st RuntimeState) {
	state.pending = st
	state.hasPending = true
	stopTimer(state.timer)
	state.timer = time.NewTimer(w.opts.Debounce)
	state.timerCh = state.timer.C
}

func (w *Writer) handleTimer(state *writerPendingState) {
	if state.hasPending {
		_ = w.writeState(state.pending)
		state.hasPending = false
	}
	state.timerCh = nil
}

func (w *Writer) handleFlush(state *writerPendingState) error {
	stopTimer(state.timer)
	state.timerCh = nil
	if !state.hasPending {
		return nil
	}
	err := w.writeState(state.pending)
	state.hasPending = false
	return err
}

func (w *Writer) handleClose(state *writerPendingState) error {
	stopTimer(state.timer)
	if !state.hasPending {
		return nil
	}
	return w.writeState(state.pending)
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func (w *Writer) writeState(state RuntimeState) error {
	state.SchemaVersion = CurrentSchemaVersion
	state.UpdatedAt = w.opts.Now()
	state.Normalize()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("sessiond: encode state: %w", err)
	}
	data = append(data, '\n')
	return SaveAtomic(w.path, data, w.opts.FileMode)
}
