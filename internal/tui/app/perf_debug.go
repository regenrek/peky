package app

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const perfDebugEnv = "PEAKYPANES_PERF_DEBUG"
const perfPaneViewAllEnv = "PEAKYPANES_PERF_PANEVIEWS_ALL"

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
	perfMu              sync.Mutex
	perfLastByKey       = map[string]time.Time{}
	perfPaneViewAllOnce sync.Once
	perfPaneViewAll     bool
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

func perfPaneViewAllEnabled() bool {
	if !perfDebugEnabled() {
		return false
	}
	perfPaneViewAllOnce.Do(func() {
		value := strings.TrimSpace(os.Getenv(perfPaneViewAllEnv))
		if value == "" {
			perfPaneViewAll = false
			return
		}
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			perfPaneViewAll = true
		default:
			perfPaneViewAll = false
		}
	})
	return perfPaneViewAll
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
