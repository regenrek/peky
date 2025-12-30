//go:build !profiler
// +build !profiler

package sessiond

func (d *Daemon) startProfiler() {
	if d == nil {
		return
	}
	_ = d.profileStop
}

func (d *Daemon) stopProfiler() {
	if d == nil {
		return
	}
	_ = d.profileStop
}
