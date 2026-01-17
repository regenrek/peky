//go:build unix

package terminal

import (
	"context"
	"log/slog"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/regenrek/peakypanes/internal/logging"
)

var ioctlSetWinsize = func(fd int, cols, rows int) error {
	return unix.IoctlSetWinsize(fd, unix.TIOCSWINSZ, &unix.Winsize{
		Row: uint16(rows), //nolint:gosec
		Col: uint16(cols), //nolint:gosec
	})
}

func setPTYSlaveWinsizeBestEffort(pty any, cols, rows int) {
	if cols <= 0 || rows <= 0 || pty == nil {
		return
	}
	slave, ok := pty.(interface{ Slave() *os.File })
	if !ok {
		return
	}
	f := slave.Slave()
	if f == nil {
		return
	}
	if err := ioctlSetWinsize(int(f.Fd()), cols, rows); err != nil {
		logging.LogEvery(
			context.Background(),
			"terminal.pty.resize.slave",
			2*time.Second,
			slog.LevelDebug,
			"terminal: pty slave winsize set failed",
			slog.Any("err", err),
			slog.Int("cols", cols),
			slog.Int("rows", rows),
		)
	}
}
