package diag

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var enabled = strings.TrimSpace(os.Getenv("PEAKYPANES_DEBUG_EVENTS")) != ""

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
	log.Printf(format, args...)
}

// LogEvery rate-limits a diagnostic log by key and interval.
func LogEvery(key string, interval time.Duration, format string, args ...any) {
	if !enabled {
		return
	}
	if interval <= 0 {
		log.Printf(format, args...)
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
	log.Printf(format, args...)
}
