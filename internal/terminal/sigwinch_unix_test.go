//go:build unix

package terminal

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func testResetKill() func() {
	prev := killProcessGroup
	return func() { killProcessGroup = prev }
}

func testResetIOCTL() func() {
	prev := ioctlGetPGRP
	return func() { ioctlGetPGRP = prev }
}

type stubPTYControl struct {
	fd  uintptr
	err error
}

func (s stubPTYControl) Control(fn func(fd uintptr)) error {
	if s.err != nil {
		return s.err
	}
	fn(s.fd)
	return nil
}

type stubPTYSlave struct {
	stubPTYControl
	slave *os.File
}

func (s stubPTYSlave) Slave() *os.File { return s.slave }

func TestSignalWINCHChoosesForegroundPGRP(t *testing.T) {
	t.Cleanup(testResetKill())
	t.Cleanup(testResetIOCTL())

	var gotPID int
	var gotSig syscall.Signal
	var calls int
	killProcessGroup = func(pid int, sig syscall.Signal) error {
		calls++
		gotPID = pid
		gotSig = sig
		return nil
	}

	ioctlGetPGRP = func(fd int) (int, error) {
		if fd != 0 {
			t.Fatalf("fd=%d", fd)
		}
		return 777, nil
	}

	signalWINCHForPTY(123, stubPTYSlave{stubPTYControl: stubPTYControl{fd: 12}, slave: os.NewFile(0, "stdin")})
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if gotPID != -777 {
		t.Fatalf("pid=%d", gotPID)
	}
	if gotSig != syscall.SIGWINCH {
		t.Fatalf("sig=%v", gotSig)
	}
}

func TestSignalWINCHFallsBackToPID(t *testing.T) {
	t.Cleanup(testResetKill())
	t.Cleanup(testResetIOCTL())

	var calls int
	var gotPID int
	killProcessGroup = func(pid int, sig syscall.Signal) error {
		calls++
		gotPID = pid
		return nil
	}

	ioctlGetPGRP = func(fd int) (int, error) {
		return 0, errors.New("nope")
	}

	signalWINCHForPTY(123, stubPTYSlave{stubPTYControl: stubPTYControl{fd: 12}, slave: os.NewFile(0, "stdin")})
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if gotPID != -123 {
		t.Fatalf("pid=%d", gotPID)
	}
}

func TestSignalWINCHSkipsInvalidPID(t *testing.T) {
	t.Cleanup(testResetKill())

	var calls int
	killProcessGroup = func(pid int, sig syscall.Signal) error {
		calls++
		return nil
	}

	signalWINCHForPTY(0, nil)
	signalWINCHForPTY(-1, nil)
	if calls != 0 {
		t.Fatalf("calls=%d", calls)
	}
}
