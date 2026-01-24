package terminal

import (
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
)

func TestPaneClosedError(t *testing.T) {
	err := &PaneClosedError{Reason: PaneClosedPTYClosed, Cause: io.EOF}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
	if !errors.Is(err, ErrPaneClosed) {
		t.Fatalf("expected ErrPaneClosed match")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected unwrap cause")
	}
}

func TestIsPTYClosedWriteError(t *testing.T) {
	cases := []error{
		syscall.EIO,
		syscall.EPIPE,
		syscall.EBADF,
		os.ErrClosed,
		io.ErrClosedPipe,
	}
	for _, err := range cases {
		if !isPTYClosedWriteError(err) {
			t.Fatalf("expected true for %v", err)
		}
	}
	if isPTYClosedWriteError(errors.New("other")) {
		t.Fatalf("expected false for other error")
	}
}
