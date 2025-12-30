//go:build !profiler
// +build !profiler

package sessiond

func (d *Daemon) startProfiler() {}

func (d *Daemon) stopProfiler() {}
