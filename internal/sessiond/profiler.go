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
	cpuProfileEnv      = "PEAKYPANES_CPU_PROFILE"
	cpuProfileSecsEnv  = "PEAKYPANES_CPU_PROFILE_SECS"
	memProfileEnv      = "PEAKYPANES_MEM_PROFILE"
	fgprofProfileEnv   = "PEAKYPANES_FGPROF"
	fgprofProfileSecs  = "PEAKYPANES_FGPROF_SECS"
	traceProfileEnv    = "PEAKYPANES_TRACE"
	traceProfileSecs   = "PEAKYPANES_TRACE_SECS"
	blockProfileEnv    = "PEAKYPANES_BLOCK_PROFILE"
	blockProfileRate   = "PEAKYPANES_BLOCK_RATE"
	mutexProfileEnv    = "PEAKYPANES_MUTEX_PROFILE"
	mutexProfileRate   = "PEAKYPANES_MUTEX_FRACTION"
	profileDoneEnv     = "PEAKYPANES_PROFILE_DONE"
	profileStartDelay  = "PEAKYPANES_PROFILE_START_DELAY"
	profileStartOnSend = "PEAKYPANES_PROFILE_START_ON_SEND"
	profileTriggerWait = "PEAKYPANES_PROFILE_TRIGGER_TIMEOUT"
	gopsEnv            = "PEAKYPANES_GOPS"
	gopsAddrEnv        = "PEAKYPANES_GOPS_ADDR"
	gopsConfigDirEnv   = "PEAKYPANES_GOPS_CONFIG_DIR"
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
			log.Printf("sessiond: profiler trigger timeout; starting anyway")
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
		if !sleepWithContext(ctx, delay) {
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
		p.stopMu.Lock()
		p.cpuFile = file
		p.stopMu.Unlock()
		log.Printf("sessiond: cpu profile started: %s", path)
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
		if !sleepWithContext(ctx, delay) {
			return
		}
		path, err := sanitizeProfilePath(p.fgPath)
		if err != nil {
			log.Printf("sessiond: fgprof profile path invalid: %v", err)
			return
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			log.Printf("sessiond: open fgprof profile: %v", err)
			return
		}
		stop := fgprof.Start(file, fgprof.FormatPprof)
		p.stopMu.Lock()
		p.fgFile = file
		p.fgStop = stop
		p.stopMu.Unlock()
		log.Printf("sessiond: fgprof profile started: %s", path)
		if !sleepWithContext(ctx, duration) {
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
		if !sleepWithContext(ctx, delay) {
			return
		}
		path, err := sanitizeProfilePath(p.tracePath)
		if err != nil {
			log.Printf("sessiond: trace profile path invalid: %v", err)
			return
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			log.Printf("sessiond: open trace profile: %v", err)
			return
		}
		if err := trace.Start(file); err != nil {
			_ = file.Close()
			log.Printf("sessiond: start trace: %v", err)
			return
		}
		p.stopMu.Lock()
		p.traceFile = file
		p.stopMu.Unlock()
		log.Printf("sessiond: trace started: %s", path)
		if !sleepWithContext(ctx, duration) {
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
			log.Printf("sessiond: fgprof stop: %v", err)
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
				log.Printf("sessiond: heap profile: %v", err)
			}
		}
		if p.blockPath != "" {
			if err := writeBlockProfile(p.blockPath); err != nil {
				log.Printf("sessiond: block profile: %v", err)
			}
		}
		if p.mutexPath != "" {
			if err := writeMutexProfile(p.mutexPath); err != nil {
				log.Printf("sessiond: mutex profile: %v", err)
			}
		}
		p.disableBlockingProfiles()
		if p.donePath != "" {
			if err := writeDoneMarker(p.donePath); err != nil {
				log.Printf("sessiond: profiler done: %v", err)
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
		log.Printf("sessiond: invalid %s: %v", env, err)
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
		log.Printf("sessiond: invalid %s: %v", env, err)
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
		log.Printf("sessiond: invalid %s: %v", env, err)
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
			log.Printf("sessiond: block profile enabled (rate=%dns)", p.blockRate)
		}
	}
	if p.mutexPath != "" {
		p.mutexRate = profileFractionFromEnv(mutexProfileRate, defaultMutexRate)
		if p.mutexRate > 0 {
			runtime.SetMutexProfileFraction(p.mutexRate)
			log.Printf("sessiond: mutex profile enabled (fraction=1/%d)", p.mutexRate)
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
			log.Printf("sessiond: close block profile: %v", cerr)
		}
	}()
	profile := pprof.Lookup("block")
	if profile == nil {
		return errors.New("block profile unavailable")
	}
	if err := profile.WriteTo(file, 0); err != nil {
		return err
	}
	log.Printf("sessiond: block profile written: %s", path)
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
			log.Printf("sessiond: close mutex profile: %v", cerr)
		}
	}()
	profile := pprof.Lookup("mutex")
	if profile == nil {
		return errors.New("mutex profile unavailable")
	}
	if err := profile.WriteTo(file, 0); err != nil {
		return err
	}
	log.Printf("sessiond: mutex profile written: %s", path)
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
	log.Printf("sessiond: profiler done: %s", path)
	return nil
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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
			log.Printf("sessiond: gops addr invalid: %v", err)
		}
	}
	if dirRaw := strings.TrimSpace(os.Getenv(gopsConfigDirEnv)); dirRaw != "" {
		if dir, err := sanitizeDirPath(dirRaw); err == nil {
			opts.ConfigDir = dir
		} else {
			log.Printf("sessiond: gops config dir invalid: %v", err)
		}
	}
	if err := agent.Listen(opts); err != nil {
		log.Printf("sessiond: gops agent: %v", err)
	}
}
