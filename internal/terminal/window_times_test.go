package terminal

import (
	"testing"
	"time"
)

func TestWindowStageTimes(t *testing.T) {
	base := time.Now().Add(-time.Second)
	w := &Window{createdAt: base}
	ptyAt := base.Add(10 * time.Millisecond)
	procAt := base.Add(20 * time.Millisecond)
	ioAt := base.Add(30 * time.Millisecond)
	updateAt := base.Add(40 * time.Millisecond)

	w.ptyCreatedAt.Store(ptyAt.UnixNano())
	w.processStartedAt.Store(procAt.UnixNano())
	w.ioStartedAt.Store(ioAt.UnixNano())
	w.firstUpdateAt.Store(updateAt.UnixNano())

	if got := w.CreatedAt(); !got.Equal(base) {
		t.Fatalf("CreatedAt mismatch: got %v want %v", got, base)
	}
	if got := w.PtyCreatedAt(); !got.Equal(ptyAt) {
		t.Fatalf("PtyCreatedAt mismatch: got %v want %v", got, ptyAt)
	}
	if got := w.ProcessStartedAt(); !got.Equal(procAt) {
		t.Fatalf("ProcessStartedAt mismatch: got %v want %v", got, procAt)
	}
	if got := w.IOStartedAt(); !got.Equal(ioAt) {
		t.Fatalf("IOStartedAt mismatch: got %v want %v", got, ioAt)
	}
	if got := w.FirstUpdateAt(); !got.Equal(updateAt) {
		t.Fatalf("FirstUpdateAt mismatch: got %v want %v", got, updateAt)
	}
}
