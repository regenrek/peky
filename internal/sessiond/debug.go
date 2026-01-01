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

const (
	perfPaneViewFirst = 1 << iota
	perfPaneViewFirstNonEmpty
	perfPaneViewFirstRequest
	perfPaneViewFirstAfterOutput
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

type paneViewTiming interface {
	CreatedAt() time.Time
	FirstReadAt() time.Time
}

func (d *Daemon) paneTimingSnapshot(paneID string) (time.Time, time.Time, bool) {
	if d == nil || paneID == "" {
		return time.Time{}, time.Time{}, false
	}
	manager, err := d.requireManager()
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	win := manager.Window(paneID)
	if win == nil {
		return time.Time{}, time.Time{}, false
	}
	timing, ok := win.(paneViewTiming)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	return timing.CreatedAt(), timing.FirstReadAt(), true
}

func (d *Daemon) logPaneViewRequest(paneID string, req PaneViewRequest) {
	if d == nil || !perfDebugEnabled() {
		return
	}
	created, firstRead, ok := d.paneTimingSnapshot(paneID)
	if !ok {
		return
	}
	now := time.Now()
	sinceStart := "n/a"
	if !created.IsZero() {
		sinceStart = now.Sub(created).String()
	}
	sinceOutput := "n/a"
	if !firstRead.IsZero() {
		sinceOutput = now.Sub(firstRead).String()
	}

	d.perfPaneViewMu.Lock()
	if d.perfPaneView == nil {
		d.perfPaneView = make(map[string]uint8)
	}
	flags := d.perfPaneView[paneID]
	if flags&perfPaneViewFirstRequest == 0 {
		flags |= perfPaneViewFirstRequest
		log.Printf("sessiond: pane view first request pane=%s mode=%v cols=%d rows=%d since_start=%s since_output=%s",
			paneID, req.Mode, req.Cols, req.Rows, sinceStart, sinceOutput)
	}
	d.perfPaneView[paneID] = flags
	d.perfPaneViewMu.Unlock()
}

func (d *Daemon) logPaneViewFirstAfterOutput(paneID string, requestAt, computeStart, renderedAt time.Time, resp PaneViewResponse) {
	if d == nil || !perfDebugEnabled() {
		return
	}
	if !paneViewAfterOutputReady(paneID, requestAt, computeStart, renderedAt, resp) {
		return
	}
	created, firstRead, ok := d.paneTimingSnapshot(paneID)
	if !ok || firstRead.IsZero() {
		return
	}
	requestAt = normalizePaneViewRequestAt(requestAt, renderedAt)
	if !d.markPerfPaneViewOnce(paneID, perfPaneViewFirstAfterOutput) {
		return
	}
	timing := buildPaneViewAfterOutputTiming(created, firstRead, requestAt, computeStart, renderedAt)
	log.Printf("sessiond: pane view first render after output pane=%s mode=%v view_bytes=%d cols=%d rows=%d since_start=%s output_to_view_req=%s view_req_to_render=%s output_to_render=%s queue_wait=%s compute=%s",
		paneID, resp.Mode, len(resp.View), resp.Cols, resp.Rows, timing.sinceStart, timing.outputToViewReq, timing.viewReqToRender, timing.outputToRender, timing.queueWait, timing.computeDur)
}

func paneViewAfterOutputReady(paneID string, requestAt, computeStart, renderedAt time.Time, resp PaneViewResponse) bool {
	if paneID == "" || renderedAt.IsZero() || computeStart.IsZero() || resp.NotModified {
		return false
	}
	return true
}

func normalizePaneViewRequestAt(requestAt, renderedAt time.Time) time.Time {
	if requestAt.IsZero() {
		return renderedAt
	}
	return requestAt
}

func (d *Daemon) markPerfPaneViewOnce(paneID string, flag uint8) bool {
	d.perfPaneViewMu.Lock()
	defer d.perfPaneViewMu.Unlock()
	if d.perfPaneView == nil {
		d.perfPaneView = make(map[string]uint8)
	}
	flags := d.perfPaneView[paneID]
	if flags&flag != 0 {
		return false
	}
	flags |= flag
	d.perfPaneView[paneID] = flags
	return true
}

type paneViewAfterOutputTiming struct {
	sinceStart      string
	outputToViewReq time.Duration
	viewReqToRender time.Duration
	outputToRender  time.Duration
	queueWait       time.Duration
	computeDur      time.Duration
}

func buildPaneViewAfterOutputTiming(created, firstRead, requestAt, computeStart, renderedAt time.Time) paneViewAfterOutputTiming {
	sinceStart := "n/a"
	if !created.IsZero() {
		sinceStart = renderedAt.Sub(created).String()
	}
	return paneViewAfterOutputTiming{
		sinceStart:      sinceStart,
		outputToViewReq: clampDuration(requestAt.Sub(firstRead)),
		viewReqToRender: clampDuration(renderedAt.Sub(requestAt)),
		outputToRender:  clampDuration(renderedAt.Sub(firstRead)),
		queueWait:       clampDuration(computeStart.Sub(requestAt)),
		computeDur:      clampDuration(renderedAt.Sub(computeStart)),
	}
}

func clampDuration(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

func (d *Daemon) logPaneViewFirst(win paneViewWindow, paneID string, mode PaneViewMode, viewLen, cols, rows int) {
	if d == nil || !perfDebugEnabled() {
		return
	}
	now := time.Now()
	var created time.Time
	var firstRead time.Time
	if timing, ok := win.(paneViewTiming); ok {
		created = timing.CreatedAt()
		firstRead = timing.FirstReadAt()
	}
	sinceStart := "n/a"
	if !created.IsZero() {
		sinceStart = now.Sub(created).String()
	}
	sinceOutput := "n/a"
	if !firstRead.IsZero() {
		sinceOutput = now.Sub(firstRead).String()
	}

	d.perfPaneViewMu.Lock()
	if d.perfPaneView == nil {
		d.perfPaneView = make(map[string]uint8)
	}
	flags := d.perfPaneView[paneID]
	if flags&perfPaneViewFirst == 0 {
		flags |= perfPaneViewFirst
		log.Printf("sessiond: pane view first render pane=%s mode=%v view_bytes=%d cols=%d rows=%d since_start=%s since_output=%s empty=%t",
			paneID, mode, viewLen, cols, rows, sinceStart, sinceOutput, viewLen == 0)
	}
	if viewLen > 0 && flags&perfPaneViewFirstNonEmpty == 0 {
		flags |= perfPaneViewFirstNonEmpty
		log.Printf("sessiond: pane view first non-empty pane=%s mode=%v view_bytes=%d cols=%d rows=%d since_start=%s since_output=%s",
			paneID, mode, viewLen, cols, rows, sinceStart, sinceOutput)
	}
	d.perfPaneView[paneID] = flags
	d.perfPaneViewMu.Unlock()
}
