//go:build !windows

package main

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestEnsureBlockingClearsNonblock(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "block")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = tmp.Close() }()

	fd := int(tmp.Fd())
	if err := unix.SetNonblock(fd, true); err != nil {
		t.Fatalf("SetNonblock: %v", err)
	}
	if err := ensureBlocking(tmp); err != nil {
		t.Fatalf("ensureBlocking: %v", err)
	}
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		t.Fatalf("F_GETFL: %v", err)
	}
	if flags&unix.O_NONBLOCK != 0 {
		t.Fatalf("expected nonblock flag cleared")
	}
}
