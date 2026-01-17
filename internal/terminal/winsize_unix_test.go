//go:build unix

package terminal

import (
	"os"
	"testing"
)

type stubPTYSlaveOnly struct {
	slave *os.File
}

func (s stubPTYSlaveOnly) Slave() *os.File { return s.slave }

func TestSetPTYSlaveWinsizeBestEffortCallsIOCTL(t *testing.T) {
	prev := ioctlSetWinsize
	t.Cleanup(func() { ioctlSetWinsize = prev })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	t.Cleanup(func() { _ = w.Close() })

	var gotFD, gotCols, gotRows int
	var calls int
	ioctlSetWinsize = func(fd int, cols, rows int) error {
		calls++
		gotFD, gotCols, gotRows = fd, cols, rows
		return nil
	}

	setPTYSlaveWinsizeBestEffort(stubPTYSlaveOnly{slave: r}, 80, 24)
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if gotFD != int(r.Fd()) {
		t.Fatalf("fd=%d", gotFD)
	}
	if gotCols != 80 || gotRows != 24 {
		t.Fatalf("cols=%d rows=%d", gotCols, gotRows)
	}
}
