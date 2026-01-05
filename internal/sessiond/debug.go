package sessiond

import (
	"context"
	"log/slog"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/termframe"
)

const (
	perfPaneViewFirst = 1 << iota
	perfPaneViewFirstNonEmpty
	perfPaneViewFirstRequest
	perfPaneViewFirstAfterOutput
)

func (d *Daemon) debugSnapshot(previewLines int, sessions []native.SessionSnapshot) {
	if d == nil || !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
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
	slog.Debug(
		"sessiond: snapshot",
		slog.Int("preview_lines", previewLines),
		slog.Int("sessions", sessionCount),
		slog.Int("panes", paneCount),
		slog.Int("dead", deadCount),
	)
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
	if d == nil || !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
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
		slog.Debug(
			"sessiond: pane view first request",
			slog.String("pane_id", paneID),
			slog.Int("cols", req.Cols),
			slog.Int("rows", req.Rows),
			slog.Bool("direct", req.DirectRender),
			slog.Any("priority", req.Priority),
			slog.String("since_start", sinceStart),
			slog.String("since_output", sinceOutput),
		)
	}
	d.perfPaneView[paneID] = flags
	d.perfPaneViewMu.Unlock()
}

func (d *Daemon) logPaneViewFirstAfterOutput(paneID string, requestAt, computeStart, renderedAt time.Time, resp PaneViewResponse) {
	if d == nil || !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
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
	slog.Debug(
		"sessiond: pane view first render after output",
		slog.String("pane_id", paneID),
		slog.Int("cell_count", len(resp.Frame.Cells)),
		slog.Int("cols", resp.Cols),
		slog.Int("rows", resp.Rows),
		slog.String("since_start", timing.sinceStart),
		slog.Duration("output_to_view_req", timing.outputToViewReq),
		slog.Duration("view_req_to_render", timing.viewReqToRender),
		slog.Duration("output_to_render", timing.outputToRender),
		slog.Duration("queue_wait", timing.queueWait),
		slog.Duration("compute", timing.computeDur),
	)
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

func (d *Daemon) logPaneViewFirst(win paneViewWindow, paneID string, cols, rows int, frame termframe.Frame) {
	if d == nil || !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
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
		slog.Debug(
			"sessiond: pane view first render",
			slog.String("pane_id", paneID),
			slog.Int("cell_count", len(frame.Cells)),
			slog.Int("cols", cols),
			slog.Int("rows", rows),
			slog.String("since_start", sinceStart),
			slog.String("since_output", sinceOutput),
			slog.Bool("empty", frame.Empty()),
		)
	}
	if !frame.Empty() && flags&perfPaneViewFirstNonEmpty == 0 {
		flags |= perfPaneViewFirstNonEmpty
		slog.Debug(
			"sessiond: pane view first non-empty",
			slog.String("pane_id", paneID),
			slog.Int("cell_count", len(frame.Cells)),
			slog.Int("cols", cols),
			slog.Int("rows", rows),
			slog.String("since_start", sinceStart),
			slog.String("since_output", sinceOutput),
		)
	}
	d.perfPaneView[paneID] = flags
	d.perfPaneViewMu.Unlock()
}
