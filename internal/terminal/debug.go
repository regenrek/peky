package terminal

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const perfDebugEnv = "PEAKYPANES_PERF_DEBUG"

const (
	perfLogInterval     = 2 * time.Second
	perfSlowLock        = 10 * time.Millisecond
	perfSlowWrite       = 15 * time.Millisecond
	perfSlowANSIRender  = 25 * time.Millisecond
	perfSlowLipgloss    = 25 * time.Millisecond
	perfSlowLipglossAll = 40 * time.Millisecond
)

var (
	perfMu        sync.Mutex
	perfLastByKey = map[string]time.Time{}
)

func perfDebugEnabled() bool {
	value := strings.TrimSpace(os.Getenv(perfDebugEnv))
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func logPerfEvery(key string, interval time.Duration, format string, args ...any) {
	if !perfDebugEnabled() {
		return
	}
	if interval <= 0 {
		log.Printf(format, args...)
		return
	}
	now := time.Now()
	perfMu.Lock()
	last := perfLastByKey[key]
	if !last.IsZero() && now.Sub(last) < interval {
		perfMu.Unlock()
		return
	}
	perfLastByKey[key] = now
	perfMu.Unlock()
	log.Printf(format, args...)
}
