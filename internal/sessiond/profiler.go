//go:build profiler
// +build profiler

package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/regenrek/peakypanes/internal/userpath"
)

const (
	cpuProfileEnv      = "PEAKYPANES_CPU_PROFILE"
	cpuProfileSecsEnv  = "PEAKYPANES_CPU_PROFILE_SECS"
	memProfileEnv      = "PEAKYPANES_MEM_PROFILE"
	defaultProfileSecs = 30
)

type daemonProfiler struct {
	cpuPath  string
	memPath  string
	cpuFile  *os.File
	stopMu   sync.Mutex
	stopOnce sync.Once
}

func (d *Daemon) startProfiler() {
	if d == nil || d.profileStop != nil {
		return
	}
	stop := startProfiler(d.ctx)
	if stop != nil {
		d.profileStop = stop
	}
}

func (d *Daemon) stopProfiler() {
	if d == nil || d.profileStop == nil {
		return
	}
	d.profileStop()
	d.profileStop = nil
}

func startProfiler(ctx context.Context) func() {
	cpuPath := strings.TrimSpace(os.Getenv(cpuProfileEnv))
	memPath := strings.TrimSpace(os.Getenv(memProfileEnv))
	if cpuPath == "" && memPath == "" {
		return nil
	}
	profiler := &daemonProfiler{
		cpuPath: cpuPath,
		memPath: memPath,
	}
	profiler.startCPU(ctx)
	profiler.stopOnContext(ctx)
	return profiler.stop
}

func (p *daemonProfiler) startCPU(ctx context.Context) {
	if p.cpuPath == "" {
		return
	}
	path, err := sanitizeProfilePath(p.cpuPath)
	if err != nil {
		log.Printf("sessiond: cpu profile path invalid: %v", err)
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		log.Printf("sessiond: open cpu profile: %v", err)
		return
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		_ = file.Close()
		log.Printf("sessiond: start cpu profile: %v", err)
		return
	}
	p.cpuFile = file
	log.Printf("sessiond: cpu profile started: %s", path)
	if duration := profileDuration(); duration > 0 {
		go p.stopAfter(ctx, duration)
	}
}

func (p *daemonProfiler) stopAfter(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		p.stop()
	case <-timer.C:
		p.stop()
	}
}

func (p *daemonProfiler) stopOnContext(ctx context.Context) {
	if ctx == nil {
		return
	}
	go func() {
		<-ctx.Done()
		p.stop()
	}()
}

func (p *daemonProfiler) stop() {
	p.stopOnce.Do(func() {
		p.stopMu.Lock()
		defer p.stopMu.Unlock()
		if p.cpuFile != nil {
			pprof.StopCPUProfile()
			_ = p.cpuFile.Close()
			p.cpuFile = nil
		}
		if p.memPath != "" {
			if err := writeHeapProfile(p.memPath); err != nil {
				log.Printf("sessiond: heap profile: %v", err)
			}
		}
	})
}

func profileDuration() time.Duration {
	raw := strings.TrimSpace(os.Getenv(cpuProfileSecsEnv))
	if raw == "" {
		return time.Duration(defaultProfileSecs) * time.Second
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	secs, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("sessiond: invalid %s: %v", cpuProfileSecsEnv, err)
		return time.Duration(defaultProfileSecs) * time.Second
	}
	return time.Duration(secs) * time.Second
}

func sanitizeProfilePath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", errors.New("profile path is required")
	}
	for _, r := range path {
		if r == 0 || unicode.IsControl(r) {
			return "", fmt.Errorf("profile path contains control characters: %q", path)
		}
	}
	path = userpath.ExpandUser(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve profile path %q: %w", path, err)
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create profile dir %q: %w", dir, err)
	}
	return abs, nil
}

func writeHeapProfile(raw string) error {
	path, err := sanitizeProfilePath(raw)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("sessiond: close heap profile: %v", cerr)
		}
	}()
	runtime.GC()
	if err := pprof.WriteHeapProfile(file); err != nil {
		return err
	}
	log.Printf("sessiond: heap profile written: %s", path)
	return nil
}
