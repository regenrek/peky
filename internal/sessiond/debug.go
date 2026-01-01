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
	if paneID == "" || renderedAt.IsZero() || computeStart.IsZero() || resp.NotModified {
		return
	}
	created, firstRead, ok := d.paneTimingSnapshot(paneID)
	if !ok || firstRead.IsZero() {
		return
	}
	if requestAt.IsZero() {
		requestAt = renderedAt
	}

	d.perfPaneViewMu.Lock()
	if d.perfPaneView == nil {
		d.perfPaneView = make(map[string]uint8)
	}
	flags := d.perfPaneView[paneID]
	if flags&perfPaneViewFirstAfterOutput != 0 {
		d.perfPaneViewMu.Unlock()
		return
	}
	flags |= perfPaneViewFirstAfterOutput
	d.perfPaneView[paneID] = flags
	d.perfPaneViewMu.Unlock()

	sinceStart := "n/a"
	if !created.IsZero() {
		sinceStart = renderedAt.Sub(created).String()
	}

	outputToViewReq := requestAt.Sub(firstRead)
	if outputToViewReq < 0 {
		outputToViewReq = 0
	}
	viewReqToRender := renderedAt.Sub(requestAt)
	if viewReqToRender < 0 {
		viewReqToRender = 0
	}
	outputToRender := renderedAt.Sub(firstRead)
	if outputToRender < 0 {
		outputToRender = 0
	}
	queueWait := computeStart.Sub(requestAt)
	if queueWait < 0 {
		queueWait = 0
	}
	computeDur := renderedAt.Sub(computeStart)
	if computeDur < 0 {
		computeDur = 0
	}

	log.Printf("sessiond: pane view first render after output pane=%s mode=%v view_bytes=%d cols=%d rows=%d since_start=%s output_to_view_req=%s view_req_to_render=%s output_to_render=%s queue_wait=%s compute=%s",
		paneID, resp.Mode, len(resp.View), resp.Cols, resp.Rows, sinceStart, outputToViewReq, viewReqToRender, outputToRender, queueWait, computeDur)
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
