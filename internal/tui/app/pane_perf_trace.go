package app

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

type paneUpdatePerf struct {
	pendingAt       time.Time
	pendingFirstSeq uint64
	pendingLastSeq  uint64
	pendingCount    int

	lastEventAt  time.Time
	lastEventSeq uint64

	lastReqAt           time.Time
	lastReqPendingAt    time.Time
	lastReqPendingCount int
	lastReqPendingSeq   uint64

	lastRespAt  time.Time
	lastRespSeq uint64
}

func (m *Model) perfEnsurePanePerf() bool {
	if m == nil || !perfDebugEnabled() {
		return false
	}
	if m.paneUpdatePerf == nil {
		m.paneUpdatePerf = make(map[string]*paneUpdatePerf)
	}
	if m.paneViewQueuedAt == nil {
		m.paneViewQueuedAt = make(map[string]time.Time)
	}
	return true
}

func (m *Model) perfPrunePanePerf(now time.Time) {
	if m == nil || !perfDebugEnabled() || m.paneUpdatePerf == nil {
		return
	}
	if !m.panePerfLastPrune.IsZero() && now.Sub(m.panePerfLastPrune) < perfPaneTraceTTL {
		return
	}
	for paneID, trace := range m.paneUpdatePerf {
		if trace == nil {
			delete(m.paneUpdatePerf, paneID)
			continue
		}
		last := trace.lastRespAt
		if last.IsZero() {
			last = trace.lastEventAt
		}
		if last.IsZero() {
			delete(m.paneUpdatePerf, paneID)
			continue
		}
		if now.Sub(last) > perfPaneTraceTTL && trace.pendingAt.IsZero() {
			delete(m.paneUpdatePerf, paneID)
		}
	}
	m.panePerfLastPrune = now
}

func (m *Model) perfNotePaneUpdated(paneID string, seq uint64, now time.Time) {
	if !m.perfEnsurePanePerf() || paneID == "" {
		return
	}
	m.perfPrunePanePerf(now)
	trace := m.paneUpdatePerf[paneID]
	if trace == nil {
		trace = &paneUpdatePerf{}
		m.paneUpdatePerf[paneID] = trace
	}
	if trace.pendingAt.IsZero() {
		trace.pendingAt = now
		trace.pendingFirstSeq = seq
		trace.pendingLastSeq = seq
		trace.pendingCount = 1
	} else {
		trace.pendingLastSeq = seq
		trace.pendingCount++
	}
	trace.lastEventAt = now
	trace.lastEventSeq = seq
}

func (m *Model) perfNotePaneQueued(paneID, reason string, now time.Time) {
	if !m.perfEnsurePanePerf() || paneID == "" {
		return
	}
	if _, ok := m.paneViewQueuedAt[paneID]; !ok {
		m.paneViewQueuedAt[paneID] = now
	}
	if reason != "" {
		logPerfEvery("tui.paneviews.queue."+paneID+"."+reason, perfLogInterval, "tui: pane view queued pane=%s reason=%s %s", paneID, reason, m.panePerfContext(paneID))
	}
}

func (m *Model) perfNotePaneQueuedBatch(ids map[string]struct{}, reason string) {
	if m == nil || len(ids) == 0 || reason == "" {
		return
	}
	now := time.Now()
	for paneID := range ids {
		if paneID == "" {
			continue
		}
		m.perfNotePaneQueued(paneID, reason, now)
	}
}

func (m *Model) perfNotePaneViewRequest(req sessiond.PaneViewRequest, now time.Time) {
	if !m.perfEnsurePanePerf() || req.PaneID == "" {
		return
	}
	trace := m.paneUpdatePerf[req.PaneID]
	if trace == nil {
		trace = &paneUpdatePerf{}
		m.paneUpdatePerf[req.PaneID] = trace
	}
	if !trace.pendingAt.IsZero() {
		delay := now.Sub(trace.pendingAt)
		if perfTraceAllEnabled() {
			slog.Debug(
				"tui: pane view request issued",
				slog.String("pane_id", req.PaneID),
				slog.Duration("event_to_req", delay),
				slog.Int("pending_count", trace.pendingCount),
				slog.Uint64("last_seq", trace.pendingLastSeq),
				slog.Int("cols", req.Cols),
				slog.Int("rows", req.Rows),
			)
		} else if delay >= perfSlowPaneEventToReq {
			logPerfEvery("tui.paneviews.req."+req.PaneID, perfLogInterval, "tui: pane view req slow pane=%s delay_event_to_req=%s pending_count=%d last_seq=%d cols=%d rows=%d", req.PaneID, delay, trace.pendingCount, trace.pendingLastSeq, req.Cols, req.Rows)
		}
		trace.lastReqPendingAt = trace.pendingAt
		trace.lastReqPendingCount = trace.pendingCount
		trace.lastReqPendingSeq = trace.pendingLastSeq
		trace.pendingAt = time.Time{}
		trace.pendingCount = 0
	} else {
		// Clear stale pending metadata so refresh-driven requests don't inherit old event timing.
		trace.lastReqPendingAt = time.Time{}
		trace.lastReqPendingCount = 0
		trace.lastReqPendingSeq = 0
	}
	trace.lastReqAt = now
	delete(m.paneViewQueuedAt, req.PaneID)
}

func (m *Model) perfNotePaneViewResponse(view sessiond.PaneViewResponse, now time.Time) {
	if !m.perfEnsurePanePerf() || view.PaneID == "" {
		return
	}
	trace := m.paneUpdatePerf[view.PaneID]
	if trace == nil {
		trace = &paneUpdatePerf{}
		m.paneUpdatePerf[view.PaneID] = trace
	}
	trace.lastRespAt = now
	trace.lastRespSeq = view.UpdateSeq

	if !trace.lastReqAt.IsZero() && !trace.lastReqPendingAt.IsZero() {
		eventToResp := now.Sub(trace.lastReqPendingAt)
		reqToResp := now.Sub(trace.lastReqAt)
		if perfTraceAllEnabled() {
			slog.Debug(
				"tui: pane view response",
				slog.String("pane_id", view.PaneID),
				slog.Duration("event_to_resp", eventToResp),
				slog.Duration("req_dur", reqToResp),
				slog.Uint64("resp_seq", view.UpdateSeq),
				slog.Uint64("last_event_seq", trace.lastEventSeq),
			)
		} else if eventToResp >= perfSlowPaneEventToResp {
			logPerfEvery("tui.paneviews.resp."+view.PaneID, perfLogInterval, "tui: pane view resp slow pane=%s delay_event_to_resp=%s delay_req_to_resp=%s resp_seq=%d last_event_seq=%d", view.PaneID, eventToResp, reqToResp, view.UpdateSeq, trace.lastEventSeq)
		}
	}
}

func (m *Model) panePerfContext(paneID string) string {
	if !perfDebugEnabled() || m == nil || paneID == "" {
		return ""
	}
	now := time.Now()
	pendingAge := time.Duration(0)
	lastSeq := uint64(0)
	if trace := m.paneUpdatePerf[paneID]; trace != nil {
		lastSeq = trace.lastEventSeq
		if !trace.pendingAt.IsZero() {
			pendingAge = now.Sub(trace.pendingAt)
		}
	}
	queuedAge := time.Duration(0)
	if queuedAt, ok := m.paneViewQueuedAt[paneID]; ok && !queuedAt.IsZero() {
		queuedAge = now.Sub(queuedAt)
	}
	if pendingAge == 0 && queuedAge == 0 && lastSeq == 0 {
		return ""
	}
	return fmt.Sprintf("pending_age=%s queued_age=%s last_event_seq=%d", pendingAge, queuedAge, lastSeq)
}
