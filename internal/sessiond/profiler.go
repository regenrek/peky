//go:build profiler
// +build profiler

package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/felixge/fgprof"
	"github.com/google/gops/agent"

	"github.com/regenrek/peakypanes/internal/profiling"
	"github.com/regenrek/peakypanes/internal/userpath"
)

const (
	cpuProfileEnv      = "PEKY_CPU_PROFILE"
	cpuProfileSecsEnv  = "PEKY_CPU_PROFILE_SECS"
	memProfileEnv      = "PEKY_MEM_PROFILE"
	fgprofProfileEnv   = "PEKY_FGPROF"
	fgprofProfileSecs  = "PEKY_FGPROF_SECS"
	traceProfileEnv    = "PEKY_TRACE"
	traceProfileSecs   = "PEKY_TRACE_SECS"
	blockProfileEnv    = "PEKY_BLOCK_PROFILE"
	blockProfileRate   = "PEKY_BLOCK_RATE"
	mutexProfileEnv    = "PEKY_MUTEX_PROFILE"
	mutexProfileRate   = "PEKY_MUTEX_FRACTION"
	profileDoneEnv     = "PEKY_PROFILE_DONE"
	profileStartDelay  = "PEKY_PROFILE_START_DELAY"
	profileStartOnSend = "PEKY_PROFILE_START_ON_SEND"
	profileTriggerWait = "PEKY_PROFILE_TRIGGER_TIMEOUT"
	gopsEnv            = "PEKY_GOPS"
	gopsAddrEnv        = "PEKY_GOPS_ADDR"
	gopsConfigDirEnv   = "PEKY_GOPS_CONFIG_DIR"
	defaultProfileSecs = 30
	defaultBlockRate   = 10 * time.Millisecond
	defaultMutexRate   = 1
)

type daemonProfiler struct {
	cpuPath    string
	memPath    string
	fgPath     string
	tracePath  string
	blockPath  string
	mutexPath  string
	donePath   string
	cpuFile    *os.File
	traceFile  *os.File
	fgFile     *os.File
	fgStop     func() error
	cpuDur     time.Duration
	fgDur      time.Duration
	traceDur   time.Duration
	startDelay time.Duration
	blockRate  int
	mutexRate  int
	stopMu     sync.Mutex
	stopOnce   sync.Once
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
	fgPath := strings.TrimSpace(os.Getenv(fgprofProfileEnv))
	tracePath := strings.TrimSpace(os.Getenv(traceProfileEnv))
	blockPath := strings.TrimSpace(os.Getenv(blockProfileEnv))
	mutexPath := strings.TrimSpace(os.Getenv(mutexProfileEnv))
	donePath := strings.TrimSpace(os.Getenv(profileDoneEnv))
	enableGops := envBool(gopsEnv)
	if cpuPath == "" && memPath == "" && fgPath == "" && tracePath == "" && blockPath == "" && mutexPath == "" && !enableGops {
		return nil
	}
	profiler := &daemonProfiler{
		cpuPath:   cpuPath,
		memPath:   memPath,
		fgPath:    fgPath,
		tracePath: tracePath,
		blockPath: blockPath,
		mutexPath: mutexPath,
		donePath:  donePath,
	}
	profiler.configureBlockingProfiles()
	if enableGops {
		profiler.startGopsAgent()
	}
	profiler.startProfiles(ctx)
	return profiler.stop
}

func (p *daemonProfiler) startProfiles(ctx context.Context) {
	if p == nil {
		return
	}
	durations := p.resolveProfileDurations()
	startOnSend := envBool(profileStartOnSend)
	p.stopOnContext(ctx)
	go p.runProfileSchedule(ctx, durations, startOnSend)
}

type profileDurations struct {
	cpuDur     time.Duration
	fgDur      time.Duration
	traceDur   time.Duration
	startDelay time.Duration
	waitFor    time.Duration
}

func (p *daemonProfiler) resolveProfileDurations() profileDurations {
	p.cpuDur = profileDuration()
	p.fgDur = profileDurationFromEnv(fgprofProfileSecs, p.cpuDur)
	p.traceDur = profileDurationFromEnv(traceProfileSecs, p.cpuDur)
	p.startDelay = profileDurationFromEnv(profileStartDelay, 0)
	return profileDurations{
		cpuDur:     p.cpuDur,
		fgDur:      p.fgDur,
		traceDur:   p.traceDur,
		startDelay: p.startDelay,
		waitFor:    profileDurationFromEnv(profileTriggerWait, 0),
	}
}

func (p *daemonProfiler) runProfileSchedule(ctx context.Context, d profileDurations, startOnSend bool) {
	if startOnSend {
		if !profiling.Wait(ctx, d.waitFor) {
			slog.Warn("sessiond: profiler trigger timeout; starting anyway")
		}
	}
	total := p.profileTotalDuration(d)
	p.startCPU(ctx, d.cpuDur, d.startDelay)
	offset := p.profileOffsets(d)
	if p.fgPath != "" && d.fgDur > 0 {
		p.startFgprof(ctx, d.fgDur, offset)
		offset += d.fgDur
	}
	if p.tracePath != "" && d.traceDur > 0 {
		p.startTrace(ctx, d.traceDur, offset)
	}
	if total > 0 {
		p.stopAfter(ctx, total)
	}
}

func (p *daemonProfiler) profileTotalDuration(d profileDurations) time.Duration {
	total := time.Duration(0)
	if d.startDelay > 0 {
		total += d.startDelay
	}
	if p.cpuPath != "" && d.cpuDur > 0 {
		total += d.cpuDur
	}
	if p.fgPath != "" && d.fgDur > 0 {
		total += d.fgDur
	}
	if p.tracePath != "" && d.traceDur > 0 {
		total += d.traceDur
	}
	return total
}

func (p *daemonProfiler) profileOffsets(d profileDurations) time.Duration {
	offset := d.startDelay
	if p.cpuPath != "" && d.cpuDur > 0 {
		offset += d.cpuDur
	}
	return offset
}

func (p *daemonProfiler) startCPU(ctx context.Context, duration, delay time.Duration) {
	if p == nil || p.cpuPath == "" {
		return
	}
	go func() {
		if err := sleepWithContext(ctx, delay); err != nil {
			return
		}
		path, err := sanitizeProfilePath(p.cpuPath)
		if err != nil {
			slog.Warn("sessiond: cpu profile path invalid", slog.Any("err", err))
			return
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			slog.Warn("sessiond: open cpu profile failed", slog.Any("err", err))
			return
		}
		if err := pprof.StartCPUProfile(file); err != nil {
			_ = file.Close()
			slog.Warn("sessiond: start cpu profile failed", slog.Any("err", err))
			return
		}
		p.stopMu.Lock()
		p.cpuFile = file
		p.stopMu.Unlock()
		slog.Info("sessiond: cpu profile started", slog.String("path", path))
		if duration <= 0 {
			return
		}
		timer := time.NewTimer(duration)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			p.stopCPU()
		case <-timer.C:
			p.stopCPU()
		}
	}()
}

func (p *daemonProfiler) startFgprof(ctx context.Context, duration, delay time.Duration) {
	if p == nil || p.fgPath == "" || duration <= 0 {
		return
	}
	go func() {
		if err := sleepWithContext(ctx, delay); err != nil {
			return
		}
		path, err := sanitizeProfilePath(p.fgPath)
		if err != nil {
			slog.Warn("sessiond: fgprof profile path invalid", slog.Any("err", err))
			return
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			slog.Warn("sessiond: open fgprof profile failed", slog.Any("err", err))
			return
		}
		stop := fgprof.Start(file, fgprof.FormatPprof)
		p.stopMu.Lock()
		p.fgFile = file
		p.fgStop = stop
		p.stopMu.Unlock()
		slog.Info("sessiond: fgprof profile started", slog.String("path", path))
		if err := sleepWithContext(ctx, duration); err != nil {
			p.stopFgprof()
			return
		}
		p.stopFgprof()
	}()
}

func (p *daemonProfiler) startTrace(ctx context.Context, duration, delay time.Duration) {
	if p == nil || p.tracePath == "" || duration <= 0 {
		return
	}
	go func() {
		if err := sleepWithContext(ctx, delay); err != nil {
			return
		}
		path, err := sanitizeProfilePath(p.tracePath)
		if err != nil {
			slog.Warn("sessiond: trace profile path invalid", slog.Any("err", err))
			return
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			slog.Warn("sessiond: open trace profile failed", slog.Any("err", err))
			return
		}
		if err := trace.Start(file); err != nil {
			_ = file.Close()
			slog.Warn("sessiond: start trace failed", slog.Any("err", err))
			return
		}
		p.stopMu.Lock()
		p.traceFile = file
		p.stopMu.Unlock()
		slog.Info("sessiond: trace started", slog.String("path", path))
		if err := sleepWithContext(ctx, duration); err != nil {
			p.stopTrace()
			return
		}
		p.stopTrace()
	}()
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

func (p *daemonProfiler) stopCPU() {
	p.stopMu.Lock()
	defer p.stopMu.Unlock()
	if p.cpuFile == nil {
		return
	}
	pprof.StopCPUProfile()
	_ = p.cpuFile.Close()
	p.cpuFile = nil
}

func (p *daemonProfiler) stopFgprof() {
	p.stopMu.Lock()
	defer p.stopMu.Unlock()
	if p.fgFile == nil {
		return
	}
	if p.fgStop != nil {
		if err := p.fgStop(); err != nil {
			slog.Warn("sessiond: fgprof stop failed", slog.Any("err", err))
		}
	}
	_ = p.fgFile.Close()
	p.fgFile = nil
	p.fgStop = nil
}

func (p *daemonProfiler) stopTrace() {
	p.stopMu.Lock()
	defer p.stopMu.Unlock()
	if p.traceFile == nil {
		return
	}
	trace.Stop()
	_ = p.traceFile.Close()
	p.traceFile = nil
}

func (p *daemonProfiler) stop() {
	p.stopOnce.Do(func() {
		p.stopCPU()
		p.stopFgprof()
		p.stopTrace()
		if p.memPath != "" {
			if err := writeHeapProfile(p.memPath); err != nil {
				slog.Warn("sessiond: heap profile failed", slog.Any("err", err))
			}
		}
		if p.blockPath != "" {
			if err := writeBlockProfile(p.blockPath); err != nil {
				slog.Warn("sessiond: block profile failed", slog.Any("err", err))
			}
		}
		if p.mutexPath != "" {
			if err := writeMutexProfile(p.mutexPath); err != nil {
				slog.Warn("sessiond: mutex profile failed", slog.Any("err", err))
			}
		}
		p.disableBlockingProfiles()
		if p.donePath != "" {
			if err := writeDoneMarker(p.donePath); err != nil {
				slog.Warn("sessiond: profiler done hook failed", slog.Any("err", err))
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
		slog.Warn("sessiond: invalid env", slog.String("env", cpuProfileSecsEnv), slog.Any("err", err))
		return time.Duration(defaultProfileSecs) * time.Second
	}
	return time.Duration(secs) * time.Second
}

func profileDurationFromEnv(env string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	secs, err := strconv.Atoi(raw)
	if err != nil {
		slog.Warn("sessiond: invalid env", slog.String("env", env), slog.Any("err", err))
		return fallback
	}
	return time.Duration(secs) * time.Second
}

func profileRateFromEnv(env string, fallback time.Duration) int {
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		if fallback <= 0 {
			return 0
		}
		return int(fallback)
	}
	if d, err := time.ParseDuration(raw); err == nil {
		if d <= 0 {
			return 0
		}
		return int(d)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		slog.Warn("sessiond: invalid env", slog.String("env", env), slog.Any("err", err))
		if fallback <= 0 {
			return 0
		}
		return int(fallback)
	}
	if value < 0 {
		return 0
	}
	return value
}

func profileFractionFromEnv(env string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		slog.Warn("sessiond: invalid env", slog.String("env", env), slog.Any("err", err))
		return fallback
	}
	if value < 1 {
		return fallback
	}
	return value
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

func (p *daemonProfiler) configureBlockingProfiles() {
	if p == nil {
		return
	}
	if p.blockPath != "" {
		p.blockRate = profileRateFromEnv(blockProfileRate, defaultBlockRate)
		if p.blockRate > 0 {
			runtime.SetBlockProfileRate(p.blockRate)
			slog.Info("sessiond: block profile enabled", slog.Int64("rate_ns", int64(p.blockRate)))
		}
	}
	if p.mutexPath != "" {
		p.mutexRate = profileFractionFromEnv(mutexProfileRate, defaultMutexRate)
		if p.mutexRate > 0 {
			runtime.SetMutexProfileFraction(p.mutexRate)
			slog.Info("sessiond: mutex profile enabled", slog.Int("fraction", p.mutexRate))
		}
	}
}

func (p *daemonProfiler) disableBlockingProfiles() {
	if p == nil {
		return
	}
	if p.blockPath != "" {
		runtime.SetBlockProfileRate(0)
	}
	if p.mutexPath != "" {
		runtime.SetMutexProfileFraction(0)
	}
}

func sanitizeDirPath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", errors.New("directory path is required")
	}
	for _, r := range path {
		if r == 0 || unicode.IsControl(r) {
			return "", fmt.Errorf("directory path contains control characters: %q", path)
		}
	}
	path = userpath.ExpandUser(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve directory path %q: %w", path, err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("create directory %q: %w", abs, err)
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
			slog.Warn("sessiond: close heap profile failed", slog.Any("err", cerr))
		}
	}()
	runtime.GC()
	if err := pprof.WriteHeapProfile(file); err != nil {
		return err
	}
	slog.Info("sessiond: heap profile written", slog.String("path", path))
	return nil
}

func writeBlockProfile(raw string) error {
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
			slog.Warn("sessiond: close block profile failed", slog.Any("err", cerr))
		}
	}()
	profile := pprof.Lookup("block")
	if profile == nil {
		return errors.New("block profile unavailable")
	}
	if err := profile.WriteTo(file, 0); err != nil {
		return err
	}
	slog.Info("sessiond: block profile written", slog.String("path", path))
	return nil
}

func writeMutexProfile(raw string) error {
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
			slog.Warn("sessiond: close mutex profile failed", slog.Any("err", cerr))
		}
	}()
	profile := pprof.Lookup("mutex")
	if profile == nil {
		return errors.New("mutex profile unavailable")
	}
	if err := profile.WriteTo(file, 0); err != nil {
		return err
	}
	slog.Info("sessiond: mutex profile written", slog.String("path", path))
	return nil
}

func writeDoneMarker(raw string) error {
	path, err := sanitizeProfilePath(raw)
	if err != nil {
		return err
	}
	payload := []byte(time.Now().Format(time.RFC3339Nano) + "\n")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return err
	}
	slog.Info("sessiond: profiler done", slog.String("path", path))
	return nil
}

func envBool(name string) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func sanitizeAddr(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("address is required")
	}
	for _, r := range value {
		if r == 0 || unicode.IsControl(r) {
			return "", fmt.Errorf("address contains control characters: %q", value)
		}
	}
	return value, nil
}

func (p *daemonProfiler) startGopsAgent() {
	opts := agent.Options{
		ShutdownCleanup: true,
	}
	if addrRaw := strings.TrimSpace(os.Getenv(gopsAddrEnv)); addrRaw != "" {
		if addr, err := sanitizeAddr(addrRaw); err == nil {
			opts.Addr = addr
		} else {
			slog.Warn("sessiond: gops addr invalid", slog.Any("err", err))
		}
	}
	if dirRaw := strings.TrimSpace(os.Getenv(gopsConfigDirEnv)); dirRaw != "" {
		if dir, err := sanitizeDirPath(dirRaw); err == nil {
			opts.ConfigDir = dir
		} else {
			slog.Warn("sessiond: gops config dir invalid", slog.Any("err", err))
		}
	}
	if err := agent.Listen(opts); err != nil {
		slog.Warn("sessiond: gops agent failed", slog.Any("err", err))
	}
}
