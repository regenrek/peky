package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/logging"
)

const perfTraceAllEnv = "PEKY_PERF_TRACE_ALL"

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

func perfDebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}

func perfTraceAllEnabled() bool {
	if !perfDebugEnabled() {
		return false
	}
	value := strings.TrimSpace(os.Getenv(perfTraceAllEnv))
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func logPerfEvery(key string, interval time.Duration, format string, args ...any) {
	if !perfDebugEnabled() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logging.LogEvery(context.Background(), key, interval, slog.LevelDebug, msg)
}
