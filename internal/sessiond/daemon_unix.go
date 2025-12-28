//go:build !windows

package sessiond

import (
	"os"
	"os/signal"
	"syscall"
)

func (d *Daemon) handleSignals() {
	if d == nil {
		return
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		_ = d.Stop()
	}()
}
