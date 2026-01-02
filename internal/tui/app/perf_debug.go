package app

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const perfDebugEnv = "PEAKYPANES_PERF_DEBUG"
const perfTraceAllEnv = "PEAKYPANES_PERF_TRACE_ALL"

const (
	perfLogInterval              = 2 * time.Second
	perfSlowRefreshTotal         = 500 * time.Millisecond
	perfSlowSnapshot             = 300 * time.Millisecond
	perfSlowBuildDashboard       = 120 * time.Millisecond
	perfSlowPaneViewBatch        = 250 * time.Millisecond
	perfSlowPaneViewReq          = 150 * time.Millisecond
	perfSlowPaneEventToReq       = 200 * time.Millisecond
	perfSlowPaneEventToResp      = 500 * time.Millisecond
	perfPaneTraceTTL             = 30 * time.Second
	perfUrgentRefreshMinInterval = 100 * time.Millisecond
)

var (
	perfMu           sync.Mutex
	perfLastByKey    = map[string]time.Time{}
	perfTraceAllOnce sync.Once
	perfTraceAll     bool
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

func perfTraceAllEnabled() bool {
	if !perfDebugEnabled() {
		return false
	}
	perfTraceAllOnce.Do(func() {
		value := strings.TrimSpace(os.Getenv(perfTraceAllEnv))
		if value == "" {
			perfTraceAll = false
			return
		}
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			perfTraceAll = true
		default:
			perfTraceAll = false
		}
	})
	return perfTraceAll
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
