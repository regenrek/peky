package diag

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var enabled = strings.TrimSpace(os.Getenv("PEAKYPANES_DEBUG_EVENTS")) != ""
var logPath = strings.TrimSpace(os.Getenv("PEAKYPANES_DEBUG_EVENTS_LOG"))
var logger = newDiagLogger()

var (
	mu        sync.Mutex
	lastByKey = map[string]time.Time{}
)

// Enabled reports whether debug diagnostics are enabled.
func Enabled() bool {
	return enabled
}

// Logf emits a diagnostic log line when enabled.
func Logf(format string, args ...any) {
	if !enabled {
		return
	}
	logger.Printf(format, args...)
}

// LogEvery rate-limits a diagnostic log by key and interval.
func LogEvery(key string, interval time.Duration, format string, args ...any) {
	if !enabled {
		return
	}
	if interval <= 0 {
		logger.Printf(format, args...)
		return
	}
	mu.Lock()
	last := lastByKey[key]
	if !last.IsZero() && time.Since(last) < interval {
		mu.Unlock()
		return
	}
	lastByKey[key] = time.Now()
	mu.Unlock()
	logger.Printf(format, args...)
}

func newDiagLogger() *log.Logger {
	if logPath != "" {
		if file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			return log.New(file, "", log.LstdFlags)
		}
	}
	return log.New(os.Stderr, "", log.LstdFlags)
}
