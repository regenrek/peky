package sessiond

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
)

const perfDebugEnv = "PEAKYPANES_PERF_DEBUG"

const perfLogInterval = 2 * time.Second

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

func (d *Daemon) debugSnapshot(previewLines int, sessions []native.SessionSnapshot) {
	if d == nil || !perfDebugEnabled() {
		return
	}
	now := time.Now().UnixNano()
	last := d.debugSnap.Load()
	if last != 0 && now-last < int64(time.Second) {
		return
	}
	if !d.debugSnap.CompareAndSwap(last, now) {
		return
	}
	sessionCount := len(sessions)
	paneCount := 0
	deadCount := 0
	for _, sess := range sessions {
		paneCount += len(sess.Panes)
		for _, pane := range sess.Panes {
			if pane.Dead {
				deadCount++
			}
		}
	}
	log.Printf("sessiond: snapshot preview_lines=%d sessions=%d panes=%d dead=%d", previewLines, sessionCount, paneCount, deadCount)
}
