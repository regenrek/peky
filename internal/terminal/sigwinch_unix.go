//go:build unix

package terminal

import (
	"context"
	"log/slog"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/regenrek/peakypanes/internal/logging"
)

var killProcessGroup = syscall.Kill

var ioctlGetPGRP = func(fd int) (int, error) {
	return unix.IoctlGetInt(fd, unix.TIOCGPGRP)
}

type ptyController interface {
	Control(fn func(fd uintptr)) error
}

func foregroundPGRP(pty any) (int, bool) {
	if pty == nil {
		return 0, false
	}
	// Prefer the slave FD: the foreground process group is associated with the
	// controlling terminal, which is the slave end of the PTY.
	if slave, ok := pty.(interface{ Slave() *os.File }); ok {
		if f := slave.Slave(); f != nil {
			pgrp, err := ioctlGetPGRP(int(f.Fd()))
			if err == nil && pgrp > 0 {
				return pgrp, true
			}
		}
	}

	ctrl, ok := pty.(ptyController)
	if !ok {
		return 0, false
	}
	pgrp := 0
	var ioctlErr error
	if err := ctrl.Control(func(fd uintptr) {
		pgrp, ioctlErr = ioctlGetPGRP(int(fd))
	}); err != nil || ioctlErr != nil {
		return 0, false
	}
	if pgrp <= 0 {
		return 0, false
	}
	return pgrp, true
}

func signalWINCHForPTY(pid int, pty any) {
	if pid <= 0 {
		return
	}
	if pgrp, ok := foregroundPGRP(pty); ok {
		logging.LogEvery(
			context.Background(),
			"terminal.winch",
			2*time.Second,
			slog.LevelDebug,
			"terminal: SIGWINCH sent",
			slog.Int("pgrp", pgrp),
			slog.Int("pid", pid),
		)
		_ = killProcessGroup(-pgrp, syscall.SIGWINCH)
		return
	}
	logging.LogEvery(
		context.Background(),
		"terminal.winch",
		2*time.Second,
		slog.LevelDebug,
		"terminal: SIGWINCH sent",
		slog.Int("pid", pid),
	)
	_ = killProcessGroup(-pid, syscall.SIGWINCH)
}
