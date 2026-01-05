package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/logging"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

const (
	paneViewTimeout      = 2 * time.Second
	paneViewFallbackCols = 80
	paneViewFallbackRows = 24
)

type paneViewKey struct {
	PaneID       string
	Cols         int
	Rows         int
	Mode         sessiond.PaneViewMode
	ShowCursor   bool
	ColorProfile termenv.Profile
}

type paneSize struct {
	Cols int
	Rows int
}

func paneViewKeyFrom(view sessiond.PaneViewResponse) paneViewKey {
	return paneViewKey{
		PaneID:       view.PaneID,
		Cols:         view.Cols,
		Rows:         view.Rows,
		Mode:         view.Mode,
		ShowCursor:   view.ShowCursor,
		ColorProfile: view.ColorProfile,
	}
}

func (m *Model) refreshPaneViewsCmd() tea.Cmd {
	if m == nil {
		return nil
	}
	if m.previewRenderMode() == PreviewRenderOff && m.state == StateDashboard {
		return nil
	}
	if !m.paneViewDataReady() {
		m.logPaneViewSkipGlobal("snapshot_empty", m.dashboardColumnsDebug())
		return nil
	}
	hits := m.paneHits()
	if len(hits) == 0 {
		m.logPaneViewSkipGlobal("no_hits", m.paneViewSkipContext())
		return nil
	}
	for _, hit := range hits {
		if hit.PaneID == "" {
			continue
		}
		m.queuePaneViewID(hit.PaneID, "refresh")
	}
	return m.schedulePaneViewPump("refresh", 0)
}

func (m *Model) refreshPaneViewFor(paneID string) tea.Cmd {
	if m == nil || paneID == "" {
		return nil
	}
	if m.previewRenderMode() == PreviewRenderOff && m.state == StateDashboard {
		return nil
	}
	if !m.paneViewDataReady() {
		m.perfNotePaneQueued(paneID, "snapshot_empty", time.Now())
		return m.requestRefreshCmdReason("paneview_snapshot_empty", true)
	}
	hit, ok := m.paneHitFor(paneID)
	if !ok {
		if !m.paneExistsInSnapshot(paneID) {
			m.perfNotePaneQueued(paneID, "pane_not_in_snapshot", time.Now())
			return m.requestRefreshCmdReason("paneview_pane_missing_snapshot", true)
		}
		if m.tab == TabProject {
			m.queuePaneViewID(paneID, "hit_not_visible")
			return m.schedulePaneViewPump("hit_not_visible", 0)
		}
		m.perfNotePaneQueued(paneID, "hit_not_visible", time.Now())
		return nil
	}
	req := m.paneViewRequestForHit(hit)
	if req == nil {
		return nil
	}
	m.queuePaneViewID(paneID, "event")
	return m.schedulePaneViewPump("event", 0)
}

func (m *Model) refreshPaneViewsForIDs(ids map[string]struct{}) tea.Cmd {
	if m == nil || len(ids) == 0 {
		return nil
	}
	if m.previewRenderMode() == PreviewRenderOff && m.state == StateDashboard {
		return nil
	}
	if !m.paneViewDataReady() {
		m.perfNotePaneQueuedBatch(ids, "snapshot_empty")
		return m.requestRefreshCmdReason("paneview_snapshot_empty", true)
	}
	return m.refreshPaneViewsForIDsWithHits(ids)
}

func (m *Model) refreshPaneViewsForIDsWithHits(ids map[string]struct{}) tea.Cmd {
	hitByID := m.paneHitsByID()
	needsRefresh := false
	for paneID := range ids {
		if paneID == "" {
			continue
		}
		if m.queuePaneViewForID(paneID, hitByID) {
			needsRefresh = true
		}
	}
	cmd := m.schedulePaneViewPump("event_batch", 0)
	if !needsRefresh {
		return cmd
	}
	return m.combinePaneViewRefresh(cmd, m.requestRefreshCmdReason("paneview_pane_missing_snapshot", true))
}

func (m *Model) queuePaneViewForID(paneID string, hitByID map[string]mouse.PaneHit) bool {
	if _, ok := hitByID[paneID]; ok {
		m.queuePaneViewID(paneID, "event")
		return false
	}
	if !m.paneExistsInSnapshot(paneID) {
		m.perfNotePaneQueued(paneID, "pane_not_in_snapshot", time.Now())
		return true
	}
	if m.tab == TabProject {
		m.queuePaneViewID(paneID, "hit_not_visible")
		return false
	}
	m.perfNotePaneQueued(paneID, "hit_not_visible", time.Now())
	return false
}

func (m *Model) combinePaneViewRefresh(cmd tea.Cmd, refreshCmd tea.Cmd) tea.Cmd {
	if refreshCmd == nil {
		return cmd
	}
	if cmd == nil {
		return refreshCmd
	}
	return tea.Batch(cmd, refreshCmd)
}

func (m *Model) paneHitFor(paneID string) (mouse.PaneHit, bool) {

	for _, hit := range m.paneHits() {
		if hit.PaneID == paneID {
			return hit, true
		}
	}
	return mouse.PaneHit{}, false
}

func (m *Model) paneViewRequests() []sessiond.PaneViewRequest {
	if !m.paneViewDataReady() {
		m.logPaneViewSkipGlobal("snapshot_empty", m.dashboardColumnsDebug())
		return nil
	}
	hits := m.paneHits()
	if len(hits) == 0 {
		m.logPaneViewSkipGlobal("no_hits", m.paneViewSkipContext())
		return nil
	}
	reqs := make([]sessiond.PaneViewRequest, 0, len(hits))
	seen := make(map[paneViewKey]struct{})
	perf := m.paneViewPerf()
	for _, hit := range hits {
		req := m.paneViewRequestForHit(hit)
		if req == nil {
			continue
		}
		key := paneViewKey{
			PaneID:       req.PaneID,
			Cols:         req.Cols,
			Rows:         req.Rows,
			Mode:         req.Mode,
			ShowCursor:   req.ShowCursor,
			ColorProfile: req.ColorProfile,
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		minInterval := paneViewMinIntervalFor(*req, perf)
		if !m.allowPaneViewRequest(key, minInterval, false) {
			continue
		}

		reqs = append(reqs, *req)
	}
	return reqs
}

func (m *Model) recordPaneSize(paneID string, cols, rows int) {
	if m == nil || paneID == "" {
		return
	}
	if cols <= 0 || rows <= 0 {
		return
	}
	if m.paneLastSize == nil {
		m.paneLastSize = make(map[string]paneSize)
	}
	m.paneLastSize[paneID] = paneSize{Cols: cols, Rows: rows}
}

func (m *Model) paneSizeForFallback(paneID string) (int, int) {
	if m == nil || paneID == "" {
		return paneViewFallbackCols, paneViewFallbackRows
	}
	if m.paneLastSize == nil {
		return paneViewFallbackCols, paneViewFallbackRows
	}
	if size, ok := m.paneLastSize[paneID]; ok {
		if size.Cols > 0 && size.Rows > 0 {
			return size.Cols, size.Rows
		}
	}
	return paneViewFallbackCols, paneViewFallbackRows
}

func (m *Model) allowFallbackRequest(paneID string, now time.Time) bool {
	if m == nil || paneID == "" {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	if m.paneLastFallback == nil {
		m.paneLastFallback = make(map[string]time.Time)
	}
	perf := m.paneViewPerf()
	last := m.paneLastFallback[paneID]
	if !last.IsZero() && now.Sub(last) < perf.FallbackMinInterval {
		return false
	}
	m.paneLastFallback[paneID] = now
	return true
}

func (m *Model) schedulePaneViewPump(reason string, delay time.Duration) tea.Cmd {
	if m == nil {
		return nil
	}
	if m.daemonDisconnected {
		return nil
	}
	if m.paneViewPumpScheduled {
		return nil
	}
	perf := m.paneViewPerf()
	if delay < 0 {
		delay = 0
	}
	if delay == 0 {
		delay = perf.PumpBaseDelay
	}
	if delay > perf.PumpMaxDelay {
		delay = perf.PumpMaxDelay
	}
	m.paneViewPumpScheduled = true
	if perfDebugEnabled() && reason != "" {
		logPerfEvery("tui.paneviews.pump."+reason, perfLogInterval, "tui: pane view pump scheduled reason=%s delay=%s", reason, delay)
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return paneViewPumpMsg{Reason: reason}
	})
}

func nextPaneViewPumpBackoff(current time.Duration, perf PaneViewPerformance) time.Duration {
	if current <= 0 {
		return perf.PumpBaseDelay
	}
	next := current * 2
	if next > perf.PumpMaxDelay {
		next = perf.PumpMaxDelay
	}
	return next
}

func (m *Model) handlePaneViewPump(msg paneViewPumpMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	m.paneViewPumpScheduled = false
	if len(m.paneViewQueuedIDs) == 0 {
		m.paneViewPumpBackoff = 0
		return nil
	}
	if m.shouldDelayPaneViewPump() {
		return m.schedulePaneViewPumpWithBackoff("in_flight")
	}
	reqs, remaining, needsRefresh := m.buildPaneViewRequestsForPending(m.paneViewPumpMaxReqs())
	m.paneViewQueuedIDs = remaining
	return m.finishPaneViewPump(reqs, needsRefresh)
}

func (m *Model) shouldDelayPaneViewPump() bool {
	perf := m.paneViewPerf()
	return m.paneViewInFlight >= perf.MaxInFlightBatches
}

func (m *Model) paneViewPumpMaxReqs() int {
	perf := m.paneViewPerf()
	maxBatches := perf.MaxInFlightBatches - m.paneViewInFlight
	if maxBatches < 1 {
		maxBatches = 1
	}
	maxReqs := maxBatches * perf.MaxBatch
	if maxReqs < 1 {
		maxReqs = perf.MaxBatch
	}
	return maxReqs
}

func (m *Model) schedulePaneViewPumpWithBackoff(reason string) tea.Cmd {
	m.paneViewPumpBackoff = nextPaneViewPumpBackoff(m.paneViewPumpBackoff, m.paneViewPerf())
	return m.schedulePaneViewPump(reason, m.paneViewPumpBackoff)
}

func (m *Model) finishPaneViewPump(reqs []sessiond.PaneViewRequest, needsRefresh bool) tea.Cmd {
	cmds := make([]tea.Cmd, 0, 3)
	if len(reqs) > 0 {
		if cmd := m.startPaneViewFetch(reqs); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if needsRefresh {
		if cmd := m.requestRefreshCmdReason("pane_view_pending_missing", true); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	cmds = m.appendPaneViewPumpFollowup(cmds)
	if len(cmds) == 0 {
		return nil
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

func (m *Model) appendPaneViewPumpFollowup(cmds []tea.Cmd) []tea.Cmd {
	if len(m.paneViewQueuedIDs) == 0 {
		m.paneViewPumpBackoff = 0
		return cmds
	}
	if m.shouldDelayPaneViewPump() {
		m.paneViewPumpBackoff = nextPaneViewPumpBackoff(m.paneViewPumpBackoff, m.paneViewPerf())
	} else {
		m.paneViewPumpBackoff = 0
	}
	if cmd := m.schedulePaneViewPump("pending", m.paneViewPumpBackoff); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return cmds
}

func (m *Model) buildPaneViewRequestsForPending(maxReqs int) ([]sessiond.PaneViewRequest, map[string]struct{}, bool) {
	if m == nil || len(m.paneViewQueuedIDs) == 0 {
		return nil, nil, false
	}
	if m.previewRenderMode() == PreviewRenderOff && m.state == StateDashboard {
		m.paneViewQueuedIDs = nil
		m.paneViewQueuedAt = nil
		return nil, nil, false
	}
	builder := newPaneViewPendingBuild(time.Now(), maxReqs, len(m.paneViewQueuedIDs))
	if !m.paneViewDataReady() {
		m.buildPendingRequestsNoSnapshot(builder)
		return builder.reqs, builder.remaining, true
	}
	hits := m.paneHitsByID()
	missing := m.collectPendingMissing(builder, hits)
	m.buildPendingRequestsForMissing(missing, hits, builder)
	return builder.reqs, builder.remaining, builder.needsRefresh
}

func (m *Model) collectPendingMissing(b *paneViewPendingBuild, hits map[string]mouse.PaneHit) []string {
	if len(hits) == 0 {
		return m.collectPendingMissingNoHits(b)
	}
	return m.collectPendingMissingWithHits(b, hits)
}

func (m *Model) collectPendingMissingWithHits(b *paneViewPendingBuild, hits map[string]mouse.PaneHit) []string {
	missing := make([]string, 0, len(m.paneViewQueuedIDs))
	// First pass: visible panes that already have hit geometry.
	for paneID, hit := range hits {
		if paneID == "" {
			continue
		}
		if _, ok := m.paneViewQueuedIDs[paneID]; !ok {
			continue
		}
		if m.deferPendingPane(paneID, b) {
			continue
		}
		if m.tryAddPaneViewRequestWithHit(paneID, hit, b) {
			continue
		}
		missing = append(missing, paneID)
	}
	// Second pass: queued panes that are not visible.
	for paneID := range m.paneViewQueuedIDs {
		if paneID == "" {
			continue
		}
		if _, ok := hits[paneID]; ok {
			continue
		}
		if m.deferPendingPane(paneID, b) {
			continue
		}
		missing = append(missing, paneID)
	}
	return missing
}

func (m *Model) collectPendingMissingNoHits(b *paneViewPendingBuild) []string {
	missing := make([]string, 0, len(m.paneViewQueuedIDs))
	for paneID := range m.paneViewQueuedIDs {
		if paneID == "" {
			continue
		}
		if m.deferPendingPane(paneID, b) {
			continue
		}
		missing = append(missing, paneID)
	}
	return missing
}

func (m *Model) deferPendingPane(paneID string, b *paneViewPendingBuild) bool {
	if b.atLimit() || m.isPaneViewInFlight(paneID) {
		b.markRemaining(paneID)
		return true
	}
	return false
}

type paneViewPendingBuild struct {
	now          time.Time
	maxReqs      int
	reqs         []sessiond.PaneViewRequest
	remaining    map[string]struct{}
	seen         map[paneViewKey]struct{}
	needsRefresh bool
}

func newPaneViewPendingBuild(now time.Time, maxReqs int, queued int) *paneViewPendingBuild {
	if queued < 1 {
		queued = 1
	}
	return &paneViewPendingBuild{
		now:       now,
		maxReqs:   maxReqs,
		reqs:      make([]sessiond.PaneViewRequest, 0, queued),
		remaining: make(map[string]struct{}),
		seen:      make(map[paneViewKey]struct{}),
	}
}

func (b *paneViewPendingBuild) atLimit() bool {
	return b.maxReqs > 0 && len(b.reqs) >= b.maxReqs
}

func (b *paneViewPendingBuild) markRemaining(paneID string) {
	if paneID == "" {
		return
	}
	b.remaining[paneID] = struct{}{}
}

func (b *paneViewPendingBuild) addRequest(req sessiond.PaneViewRequest, key paneViewKey) bool {
	if _, ok := b.seen[key]; ok {
		return false
	}
	b.seen[key] = struct{}{}
	b.reqs = append(b.reqs, req)
	return true
}

func (m *Model) buildPendingRequestsNoSnapshot(b *paneViewPendingBuild) {
	for paneID := range m.paneViewQueuedIDs {
		if b.atLimit() || m.isPaneViewInFlight(paneID) {
			b.markRemaining(paneID)
			continue
		}
		if fallback := m.buildFallbackRequest(paneID, b.seen, b.now); fallback != nil {
			key := paneViewKey{
				PaneID:       fallback.PaneID,
				Cols:         fallback.Cols,
				Rows:         fallback.Rows,
				Mode:         fallback.Mode,
				ShowCursor:   fallback.ShowCursor,
				ColorProfile: fallback.ColorProfile,
			}
			b.addRequest(*fallback, key)
		}
		b.markRemaining(paneID)
		m.perfNotePaneQueued(paneID, "snapshot_empty", b.now)
	}
}

func (m *Model) queueFallbackAndNote(paneID string, b *paneViewPendingBuild, reason string) {
	if fallback := m.buildFallbackRequest(paneID, b.seen, b.now); fallback != nil {
		key := paneViewKey{
			PaneID:       fallback.PaneID,
			Cols:         fallback.Cols,
			Rows:         fallback.Rows,
			Mode:         fallback.Mode,
			ShowCursor:   fallback.ShowCursor,
			ColorProfile: fallback.ColorProfile,
		}
		b.addRequest(*fallback, key)
	}
	b.markRemaining(paneID)
	m.perfNotePaneQueued(paneID, reason, b.now)
}

func (m *Model) paneHitsByID() map[string]mouse.PaneHit {
	hits := m.paneHits()
	if len(hits) == 0 {
		return nil
	}
	hitByID := make(map[string]mouse.PaneHit, len(hits))
	for _, hit := range hits {
		if hit.PaneID == "" {
			continue
		}
		hitByID[hit.PaneID] = hit
	}
	return hitByID
}

func (m *Model) tryAddPaneViewRequestWithHit(paneID string, hit mouse.PaneHit, b *paneViewPendingBuild) bool {
	req := m.paneViewRequestForHit(hit)
	if req == nil {
		return false
	}
	key := paneViewKey{
		PaneID:       req.PaneID,
		Cols:         req.Cols,
		Rows:         req.Rows,
		Mode:         req.Mode,
		ShowCursor:   req.ShowCursor,
		ColorProfile: req.ColorProfile,
	}
	if _, seen := b.seen[key]; seen {
		return true
	}
	minInterval, force := m.paneViewRequestPolicy(paneID, *req)
	if !m.allowPaneViewRequest(key, minInterval, force) {
		b.markRemaining(paneID)
		return true
	}
	b.addRequest(*req, key)
	return true
}

func (m *Model) buildPendingRequestsForMissing(missing []string, hits map[string]mouse.PaneHit, b *paneViewPendingBuild) {
	if len(missing) == 0 || b == nil {
		return
	}
	// If hits are available, prioritize visible panes and keep non-visible pending
	// without emitting fallback requests that would crowd the queue.
	if len(hits) > 0 {
		for _, paneID := range missing {
			if paneID == "" {
				continue
			}
			if b.atLimit() || m.isPaneViewInFlight(paneID) {
				b.markRemaining(paneID)
				continue
			}
			if !m.paneExistsInSnapshot(paneID) {
				m.queueFallbackAndNote(paneID, b, "pane_not_in_snapshot")
				b.needsRefresh = true
				continue
			}
			b.markRemaining(paneID)
			m.perfNotePaneQueued(paneID, "hit_not_visible", b.now)
		}
		return
	}

	// No hits at all: emit fallback requests so panes can render while layout settles.
	for _, paneID := range missing {
		if paneID == "" {
			continue
		}
		if b.atLimit() || m.isPaneViewInFlight(paneID) {
			b.markRemaining(paneID)
			continue
		}
		if !m.paneExistsInSnapshot(paneID) {
			m.queueFallbackAndNote(paneID, b, "pane_not_in_snapshot")
			b.needsRefresh = true
			continue
		}
		m.queueFallbackAndNote(paneID, b, "hit_not_found")
	}
}

func (m *Model) paneViewRequestPolicy(paneID string, req sessiond.PaneViewRequest) (time.Duration, bool) {
	perf := m.paneViewPerf()
	minInterval := paneViewMinIntervalFor(req, perf)
	force := m.paneViewQueuedAge(paneID) >= perf.ForceAfter
	if !m.hasPaneViewFirst(paneID) {
		minInterval = 0
		force = true
	}
	return minInterval, force
}

func (m *Model) buildFallbackRequest(paneID string, seen map[paneViewKey]struct{}, now time.Time) *sessiond.PaneViewRequest {
	if m == nil || paneID == "" {
		return nil
	}
	if seen == nil {
		seen = make(map[paneViewKey]struct{})
	}
	if !m.allowFallbackRequest(paneID, now) {
		return nil
	}
	if m.previewRenderMode() == PreviewRenderOff && m.state == StateDashboard {
		m.logPaneViewSkip(paneID, "preview_render_off", m.paneViewSkipContext())
		return nil
	}
	cols, rows := m.paneSizeForFallback(paneID)
	req := sessiond.PaneViewRequest{
		PaneID:       paneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         sessiond.PaneViewANSI,
		ShowCursor:   false,
		ColorProfile: m.paneViewProfile,
		Priority:     sessiond.PaneViewPriorityBackground,
	}
	if m.previewRenderMode() == PreviewRenderDirect {
		req.DirectRender = true
	}
	key := paneViewKey{
		PaneID:       req.PaneID,
		Cols:         req.Cols,
		Rows:         req.Rows,
		Mode:         req.Mode,
		ShowCursor:   req.ShowCursor,
		ColorProfile: req.ColorProfile,
	}
	if _, ok := seen[key]; ok {
		return nil
	}
	if m.paneViewSeq != nil {
		req.KnownSeq = m.paneViewSeq[key]
	}
	perf := m.paneViewPerf()
	minInterval := perf.MinIntervalBackground
	force := false
	if !m.hasPaneViewFirst(paneID) {
		minInterval = 0
		force = true
	}
	if !m.allowPaneViewRequest(key, minInterval, force) {
		return nil
	}
	seen[key] = struct{}{}
	return &req
}

func (m *Model) ensurePaneViewInFlightMap() {
	if m == nil {
		return
	}
	if m.paneViewInFlightByPane == nil {
		m.paneViewInFlightByPane = make(map[string]struct{})
	}
}

func (m *Model) markPaneViewInFlight(reqs []sessiond.PaneViewRequest) {
	if m == nil || len(reqs) == 0 {
		return
	}
	m.ensurePaneViewInFlightMap()
	for _, req := range reqs {
		if req.PaneID == "" {
			continue
		}
		m.paneViewInFlightByPane[req.PaneID] = struct{}{}
	}
}

func (m *Model) clearPaneViewInFlight(paneIDs []string) {
	if m == nil || len(paneIDs) == 0 || m.paneViewInFlightByPane == nil {
		return
	}
	for _, paneID := range paneIDs {
		if paneID == "" {
			continue
		}
		delete(m.paneViewInFlightByPane, paneID)
	}
}

func (m *Model) isPaneViewInFlight(paneID string) bool {
	if m == nil || paneID == "" || m.paneViewInFlightByPane == nil {
		return false
	}
	_, ok := m.paneViewInFlightByPane[paneID]
	return ok
}

func (m *Model) hasPaneViewFirst(paneID string) bool {
	if m == nil || paneID == "" || m.paneViewFirst == nil {
		return false
	}
	_, ok := m.paneViewFirst[paneID]
	return ok
}

func (m *Model) startPaneViewFetch(reqs []sessiond.PaneViewRequest) tea.Cmd {
	if m == nil || m.client == nil || len(reqs) == 0 {
		return nil
	}
	if m.daemonDisconnected {
		return nil
	}
	now := time.Now()
	for _, req := range reqs {
		m.perfNotePaneViewRequest(req, now)
	}
	m.markPaneViewInFlight(reqs)
	perfBurst := m.perfPaneViewInitialBurst()
	if perfBurst {
		m.paneViewPerfBurstDone = true
		logPerfEvery("tui.paneviews.perf_burst", 0, "tui: pane views perf initial burst reqs=%d", len(reqs))
	}
	perf := m.paneViewPerf()
	chunks := chunkPaneViewRequests(reqs, perf.MaxBatch)
	if len(chunks) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		m.paneViewInFlight++
		cmds = append(cmds, m.fetchPaneViewsCmd(chunk))
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

func (m *Model) queuePaneViewID(paneID, reason string) {
	if m == nil || paneID == "" {
		return
	}
	if m.paneViewQueuedIDs == nil {
		m.paneViewQueuedIDs = make(map[string]struct{})
	}
	m.paneViewQueuedIDs[paneID] = struct{}{}
	if m.paneViewQueuedAt == nil {
		m.paneViewQueuedAt = make(map[string]time.Time)
	}
	if _, ok := m.paneViewQueuedAt[paneID]; !ok {
		m.paneViewQueuedAt[paneID] = time.Now()
	}
	if reason != "" {
		m.perfNotePaneQueued(paneID, reason, time.Now())
	}
}

func (m *Model) paneViewQueuedAge(paneID string) time.Duration {
	if m == nil || paneID == "" {
		return 0
	}
	if m.paneViewQueuedAt == nil {
		return 0
	}
	queuedAt, ok := m.paneViewQueuedAt[paneID]
	if !ok || queuedAt.IsZero() {
		return 0
	}
	return time.Since(queuedAt)
}

func chunkPaneViewRequests(reqs []sessiond.PaneViewRequest, maxBatch int) [][]sessiond.PaneViewRequest {
	if len(reqs) == 0 {
		return nil
	}
	if maxBatch <= 0 {
		maxBatch = len(reqs)
	}
	chunks := make([][]sessiond.PaneViewRequest, 0, (len(reqs)+maxBatch-1)/maxBatch)
	for i := 0; i < len(reqs); i += maxBatch {
		end := i + maxBatch
		if end > len(reqs) {
			end = len(reqs)
		}
		chunks = append(chunks, reqs[i:end])
	}
	return chunks
}

func (m *Model) paneViewRequestForHit(hit mouse.PaneHit) *sessiond.PaneViewRequest {
	content, ok := m.paneViewContent(hit)
	if !ok {
		return nil
	}
	cols, rows := content.W, content.H
	if cols <= 0 || rows <= 0 {
		m.logPaneViewSkip(hit.PaneID, "invalid_size", fmt.Sprintf("cols=%d rows=%d %s", cols, rows, m.paneViewSkipContext()))
		return nil
	}
	mode, showCursor, priority := m.paneViewRenderSettings(hit.PaneID)
	req := m.newPaneViewRequest(hit.PaneID, cols, rows, mode, showCursor, priority)
	m.applyKnownPaneViewSeq(req)
	return req
}

func (m *Model) paneViewContent(hit mouse.PaneHit) (mouse.Rect, bool) {
	if hit.PaneID == "" {
		m.logPaneViewSkipGlobal("missing_pane_id", m.paneViewSkipContext())
		return mouse.Rect{}, false
	}
	if m.previewRenderMode() == PreviewRenderOff && m.state == StateDashboard {
		m.logPaneViewSkip(hit.PaneID, "preview_render_off", m.paneViewSkipContext())
		return mouse.Rect{}, false
	}
	content := hit.Content
	if content.Empty() {
		if m.renderPolicyAll() && !hit.Outer.Empty() {
			content = hit.Outer
		} else {
			m.logPaneViewSkip(hit.PaneID, "content_empty", m.paneViewSkipContext())
			return mouse.Rect{}, false
		}
	}
	return content, true
}

func (m *Model) paneViewRenderSettings(paneID string) (sessiond.PaneViewMode, bool, sessiond.PaneViewPriority) {
	mode := sessiond.PaneViewANSI
	showCursor := false
	priority := sessiond.PaneViewPriorityBackground

	isSelected := m.selectedPaneID() == paneID
	if isSelected {
		priority = sessiond.PaneViewPriorityNormal
	}
	if pane := m.paneByID(paneID); pane != nil && pane.Disconnected {
		return mode, false, priority
	}
	if isSelected && m.supportsTerminalFocus() {
		mode = sessiond.PaneViewLipgloss
		showCursor = m.terminalFocus
		if showCursor {
			priority = sessiond.PaneViewPriorityFocused
		}
	}
	return mode, showCursor, priority
}

func (m *Model) selectedPaneID() string {
	if pane := m.selectedPane(); pane != nil {
		return pane.ID
	}
	return ""
}

func (m *Model) newPaneViewRequest(
	paneID string,
	cols int,
	rows int,
	mode sessiond.PaneViewMode,
	showCursor bool,
	priority sessiond.PaneViewPriority,
) *sessiond.PaneViewRequest {
	req := &sessiond.PaneViewRequest{
		PaneID:       paneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         mode,
		ShowCursor:   showCursor,
		ColorProfile: m.paneViewProfile,
		Priority:     priority,
	}
	if mode == sessiond.PaneViewANSI && m.previewRenderMode() == PreviewRenderDirect {
		req.DirectRender = true
	}
	return req
}

func (m *Model) applyKnownPaneViewSeq(req *sessiond.PaneViewRequest) {
	if req == nil || m.paneViewSeq == nil {
		return
	}
	key := paneViewKey{
		PaneID:       req.PaneID,
		Cols:         req.Cols,
		Rows:         req.Rows,
		Mode:         req.Mode,
		ShowCursor:   req.ShowCursor,
		ColorProfile: req.ColorProfile,
	}
	req.KnownSeq = m.paneViewSeq[key]
}

func (m *Model) allowPaneViewRequest(key paneViewKey, minInterval time.Duration, force bool) bool {
	if m == nil || key.PaneID == "" {
		return false
	}
	if m.paneViewLastReq == nil {
		m.paneViewLastReq = make(map[paneViewKey]time.Time)
	}
	now := time.Now()
	if last, ok := m.paneViewLastReq[key]; ok {
		if !force && minInterval > 0 && now.Sub(last) < minInterval {
			logging.LogEvery(
				context.Background(),
				"tui.paneviews.throttle",
				2*time.Second,
				slog.LevelDebug,
				"tui: pane view throttled",
				slog.String("pane_id", key.PaneID),
				slog.Duration("interval", minInterval),
			)
			m.logPaneViewSkip(key.PaneID, "throttled", fmt.Sprintf("interval=%s %s", minInterval, m.paneViewSkipContext()))
			return false
		}
	}
	m.paneViewLastReq[key] = now
	return true
}

func (m *Model) paneViewDataReady() bool {
	if m == nil {
		return false
	}
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			if session.Status == StatusStopped {
				continue
			}
			if len(session.Panes) > 0 {
				return true
			}
		}
	}
	return false
}

func (m *Model) paneViewPerf() PaneViewPerformance {
	if m == nil {
		return paneViewPerfMax
	}
	perf := m.settings.Performance.PaneViews
	if perf.MaxConcurrency <= 0 || perf.MaxBatch <= 0 || perf.MaxInFlightBatches <= 0 {
		return paneViewPerfMax
	}
	return perf
}

func (m *Model) renderPolicyAll() bool {
	if m == nil {
		return false
	}
	return m.settings.Performance.RenderPolicy == RenderPolicyAll
}

func (m *Model) previewRenderMode() string {
	if m == nil {
		return PreviewRenderDirect
	}
	mode := strings.ToLower(strings.TrimSpace(m.settings.Performance.PreviewRender.Mode))
	if mode == "" {
		return PreviewRenderDirect
	}
	return mode
}

func (m *Model) paneExistsInSnapshot(paneID string) bool {
	if m == nil || paneID == "" {
		return false
	}
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			if session.Status == StatusStopped {
				continue
			}
			for _, pane := range session.Panes {
				if pane.ID == paneID {
					return true
				}
			}
		}
	}
	return false
}

func (m *Model) perfPaneViewInitialBurst() bool {
	if m == nil {
		return false
	}
	return m.renderPolicyAll() && !m.paneViewPerfBurstDone
}

func (m *Model) paneViewSkipContext() string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("state=%d tab=%d", m.state, m.tab)
}

func (m *Model) logPaneViewSkip(paneID, reason, detail string) {
	if m == nil || !perfDebugEnabled() {
		return
	}
	if paneID == "" {
		paneID = "unknown"
	}
	if m.paneViewSkipLog == nil {
		m.paneViewSkipLog = make(map[string]struct{})
	}
	key := paneID + "|" + reason
	if _, ok := m.paneViewSkipLog[key]; ok {
		return
	}
	m.paneViewSkipLog[key] = struct{}{}
	msg := fmt.Sprintf("tui: pane view skip pane=%s reason=%s", paneID, reason)
	if detail != "" {
		msg += " " + detail
	}
	if perfCtx := m.panePerfContext(paneID); perfCtx != "" {
		msg += " " + perfCtx
	}
	logPerfEvery("tui.paneviews.skip."+key, 0, "%s", msg)
}

func (m *Model) logPaneViewSkipGlobal(reason, detail string) {
	if m == nil || !perfDebugEnabled() {
		return
	}
	if m.paneViewSkipLog == nil {
		m.paneViewSkipLog = make(map[string]struct{})
	}
	key := "global|" + reason
	if _, ok := m.paneViewSkipLog[key]; ok {
		return
	}
	m.paneViewSkipLog[key] = struct{}{}
	msg := fmt.Sprintf("tui: pane view skip-global reason=%s", reason)
	if detail != "" {
		msg += " " + detail
	}
	logPerfEvery("tui.paneviews.skip."+key, 0, "%s", msg)
}

func (m *Model) dashboardColumnsDebug() string {
	if m == nil {
		return ""
	}
	projectCount := len(m.data.Projects)
	sessionCount := 0
	paneCount := 0
	runningSessions := 0
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			sessionCount++
			if session.Status != StatusStopped {
				runningSessions++
			}
			paneCount += len(session.Panes)
		}
	}
	selectedProject := "none"
	if proj := m.selectedProject(); proj != nil {
		selectedProject = proj.ID
	}
	selectedSession := "none"
	if sess := m.selectedSession(); sess != nil {
		selectedSession = sess.Name
	}
	return fmt.Sprintf("projects=%d sessions=%d panes=%d running=%d selected_project=%s selected_session=%s state=%d tab=%d",
		projectCount, sessionCount, paneCount, runningSessions, selectedProject, selectedSession, m.state, m.tab)
}

func paneViewMinIntervalFor(req sessiond.PaneViewRequest, perf PaneViewPerformance) time.Duration {
	if req.Priority == sessiond.PaneViewPriorityFocused || req.Mode == sessiond.PaneViewLipgloss || req.ShowCursor {
		return perf.MinIntervalFocused
	}
	if req.Priority == sessiond.PaneViewPriorityNormal || req.Priority == sessiond.PaneViewPriorityUnset {
		return perf.MinIntervalSelected
	}
	return perf.MinIntervalBackground
}

func paneViewTimeoutFor(req sessiond.PaneViewRequest, perf PaneViewPerformance) time.Duration {
	if req.Priority == sessiond.PaneViewPriorityFocused || req.Mode == sessiond.PaneViewLipgloss || req.ShowCursor {
		return perf.TimeoutFocused
	}
	if req.Priority == sessiond.PaneViewPriorityNormal || req.Priority == sessiond.PaneViewPriorityUnset {
		return perf.TimeoutSelected
	}
	return perf.TimeoutBackground
}

func (m *Model) fetchPaneViewsCmd(reqs []sessiond.PaneViewRequest) tea.Cmd {
	if m == nil || len(reqs) == 0 {
		return nil
	}
	client := m.paneViewClient
	if client == nil {
		client = m.client
	}
	if client == nil {
		return nil
	}
	perf := m.paneViewPerf()
	return func() tea.Msg {
		return fetchPaneViews(client, reqs, perf)
	}
}

type paneViewResult struct {
	view sessiond.PaneViewResponse
	err  error
}

func paneViewReqPaneIDs(reqs []sessiond.PaneViewRequest) []string {
	if len(reqs) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(reqs))
	for _, req := range reqs {
		if req.PaneID == "" {
			continue
		}
		if _, ok := seen[req.PaneID]; ok {
			continue
		}
		seen[req.PaneID] = struct{}{}
		out = append(out, req.PaneID)
	}
	return out
}

func fetchPaneViews(client *sessiond.Client, reqs []sessiond.PaneViewRequest, perf PaneViewPerformance) paneViewsMsg {
	paneIDs := paneViewReqPaneIDs(reqs)
	start := time.Now()
	results := make(chan paneViewResult, len(reqs))
	jobs := make(chan sessiond.PaneViewRequest)

	workers := paneViewWorkerCount(reqs, perf)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go paneViewWorker(client, jobs, results, &wg, perf)
	}

	go func() {
		for _, req := range reqs {
			jobs <- req
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	views, firstErr := collectPaneViewResults(results)
	if perfDebugEnabled() {
		dur := time.Since(start)
		if dur > perfSlowPaneViewBatch {
			logPerfEvery("tui.paneviews.batch", perfLogInterval, "tui: pane view batch slow dur=%s reqs=%d views=%d err=%v", dur, len(reqs), len(views), firstErr)
		}
	}
	if len(views) == 0 && firstErr != nil {
		return paneViewsMsg{Err: firstErr, PaneIDs: paneIDs}
	}
	return paneViewsMsg{Views: views, Err: firstErr, PaneIDs: paneIDs}
}

func paneViewWorkerCount(reqs []sessiond.PaneViewRequest, perf PaneViewPerformance) int {
	workers := perf.MaxConcurrency
	if len(reqs) < workers {
		workers = len(reqs)
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}

func paneViewWorker(client *sessiond.Client, jobs <-chan sessiond.PaneViewRequest, results chan<- paneViewResult, wg *sync.WaitGroup, perf PaneViewPerformance) {
	defer wg.Done()
	for req := range jobs {
		timeout := paneViewTimeoutFor(req, perf)
		if timeout <= 0 {
			timeout = paneViewTimeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		if deadline, ok := ctx.Deadline(); ok {
			req.DeadlineUnixNano = deadline.UnixNano()
		}
		start := time.Now()
		resp, err := client.GetPaneView(ctx, req)
		cancel()
		if perfDebugEnabled() {
			dur := time.Since(start)
			if dur > perfSlowPaneViewReq {
				logPerfEvery("tui.paneviews.req", perfLogInterval, "tui: pane view req slow pane=%s dur=%s cols=%d rows=%d mode=%v priority=%v err=%v", req.PaneID, dur, req.Cols, req.Rows, req.Mode, req.Priority, err)
			}
		}
		if err != nil {
			results <- paneViewResult{err: err}
			continue
		}
		results <- paneViewResult{view: resp}
	}
}

func collectPaneViewResults(results <-chan paneViewResult) ([]sessiond.PaneViewResponse, error) {
	views := make([]sessiond.PaneViewResponse, 0)
	var firstErr error
	for res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		views = append(views, res.view)
	}
	return views, firstErr
}

func (m Model) paneView(paneID string, cols, rows int, mode sessiond.PaneViewMode, showCursor bool, profile termenv.Profile) string {
	if paneID == "" || cols <= 0 || rows <= 0 {
		return ""
	}
	key := paneViewKey{
		PaneID:       paneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         mode,
		ShowCursor:   showCursor,
		ColorProfile: profile,
	}
	if m.paneViews == nil {
		return ""
	}
	if view, ok := m.paneViews[key]; ok {
		return view
	}
	// Fallback to any cached view with matching pane/size when mode or cursor differs.
	for cachedKey, cachedView := range m.paneViews {
		if cachedKey.PaneID != paneID {
			continue
		}
		if cachedKey.Cols != cols || cachedKey.Rows != rows {
			continue
		}
		if cachedKey.ColorProfile != profile {
			continue
		}
		if cachedView != "" {
			return cachedView
		}
	}
	return ""
}

func detectPaneViewProfile() termenv.Profile {
	return termenv.EnvColorProfile()
}
