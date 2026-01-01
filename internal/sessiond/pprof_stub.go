//go:build !profiler
// +build !profiler

package sessiond

import (
	"errors"
	"strings"
)

func (d *Daemon) startPprofServer() error {
	if d == nil {
		return nil
	}
	if strings.TrimSpace(d.pprofAddr) == "" {
		return nil
	}
	return errors.New("sessiond: pprof requires a profiler build (use -tags profiler)")
}

func (d *Daemon) stopPprofServer() {
	if d == nil {
		return
	}
	_ = d.pprofAddr
	d.pprofServer = nil
	d.pprofListener = nil
}
