package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/regenrek/peakypanes/internal/logging"
	"github.com/regenrek/peakypanes/internal/termframe"
)

const (
	paneViewMaxConcurrency   = 4
	paneViewDeadlineSlack    = 50 * time.Millisecond
	paneViewStarvationWindow = 750 * time.Millisecond
	paneViewSlowThreshold    = 50 * time.Millisecond

	paneViewCacheTTL        = 30 * time.Second
	paneViewCacheMaxEntries = 100
)

func paneViewEffectiveSeq(win paneViewWindow) uint64 {
	if win == nil {
		return 0
	}
	if seq := win.FrameCacheSeq(); seq != 0 {
		return seq
	}
	return win.UpdateSeq()
}

type paneViewJob struct {
	env      Envelope
	req      PaneViewRequest
	seq      uint64
	received time.Time
}

type paneViewScheduler struct {
	mu       sync.Mutex
	cond     *sync.Cond
	pending  map[string]paneViewJob
	inflight map[string]bool
	cancel   map[string]context.CancelFunc
	seq      map[string]uint64
	closed   bool
}

func newPaneViewScheduler() *paneViewScheduler {
	s := &paneViewScheduler{
		pending:  make(map[string]paneViewJob),
		inflight: make(map[string]bool),
		cancel:   make(map[string]context.CancelFunc),
		seq:      make(map[string]uint64),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *paneViewScheduler) enqueue(paneID string, env Envelope, req PaneViewRequest) {
	if s == nil {
		return
	}
	var cancel context.CancelFunc
	s.mu.Lock()
	received := time.Now()
	if existing, ok := s.pending[paneID]; ok && !s.inflight[paneID] {
		received = existing.received
	}
	next := s.seq[paneID] + 1
	s.seq[paneID] = next
	s.pending[paneID] = paneViewJob{env: env, req: req, seq: next, received: received}
	cancel = s.cancel[paneID]
	s.cond.Signal()
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *paneViewScheduler) next() (string, paneViewJob, bool) {
	if s == nil {
		return "", paneViewJob{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if s.closed {
			return "", paneViewJob{}, false
		}
		paneID, job, ok := s.pickLocked()
		if ok {
			s.inflight[paneID] = true
			delete(s.pending, paneID)
			return paneID, job, true
		}
		s.cond.Wait()
	}
}

func (s *paneViewScheduler) pickLocked() (string, paneViewJob, bool) {
	var (
		bestID  string
		bestJob paneViewJob
		hasBest bool
	)

	for paneID, job := range s.pending {
		if s.inflight[paneID] {
			continue
		}
		if !hasBest {
			bestID, bestJob, hasBest = paneID, job, true
			continue
		}
		if paneViewJobBetter(job, bestJob) {
			bestID, bestJob = paneID, job
		}
	}
	if !hasBest {
		return "", paneViewJob{}, false
	}
	return bestID, bestJob, true
}

func paneViewJobBetter(a, b paneViewJob) bool {
	ap := paneViewJobPriority(a.req)
	bp := paneViewJobPriority(b.req)
	ap = paneViewJobBoost(ap, a.received)
	bp = paneViewJobBoost(bp, b.received)
	if ap != bp {
		return ap > bp
	}

	// Earlier deadline first (0 means no deadline and should be treated as far future).
	ad := a.req.DeadlineUnixNano
	bd := b.req.DeadlineUnixNano
	const maxI64 = int64(^uint64(0) >> 1)
	if ad <= 0 {
		ad = maxI64
	}
	if bd <= 0 {
		bd = maxI64
	}
	if ad != bd {
		return ad < bd
	}

	// Oldest received first.
	return a.received.Before(b.received)
}

func paneViewJobPriority(req PaneViewRequest) int {
	if req.Priority != PaneViewPriorityUnset {
		return int(req.Priority)
	}
	return int(PaneViewPriorityNormal)
}

func paneViewJobBoost(priority int, received time.Time) int {
	if received.IsZero() {
		return priority
	}
	age := time.Since(received)
	if age < paneViewStarvationWindow {
		return priority
	}
	boost := int(age / paneViewStarvationWindow)
	out := priority + boost
	if out > int(PaneViewPriorityFocused) {
		return int(PaneViewPriorityFocused)
	}
	return out
}

func (s *paneViewScheduler) finish(paneID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.inflight, paneID)
	delete(s.cancel, paneID)
	s.cond.Signal()
	s.mu.Unlock()
}

func (s *paneViewScheduler) setCancel(paneID string, cancel context.CancelFunc) {
	if s == nil || paneID == "" {
		return
	}
	s.mu.Lock()
	s.cancel[paneID] = cancel
	s.mu.Unlock()
}

func (s *paneViewScheduler) clearCancel(paneID string) {
	if s == nil || paneID == "" {
		return
	}
	s.mu.Lock()
	delete(s.cancel, paneID)
	s.mu.Unlock()
}

func (s *paneViewScheduler) isLatest(paneID string, seq uint64) bool {
	if s == nil {
		return true
	}
	s.mu.Lock()
	latest := s.seq[paneID]
	s.mu.Unlock()
	return seq == latest
}

func (s *paneViewScheduler) close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	for _, cancel := range s.cancel {
		if cancel != nil {
			cancel()
		}
	}
	s.cond.Broadcast()
	s.mu.Unlock()
}

type paneViewCacheKey struct {
	PaneID string
	Cols   int
	Rows   int
}

type cachedPaneView struct {
	resp       PaneViewResponse
	renderedAt time.Time
	updateSeq  uint64
}

func paneViewCacheKeyFor(paneID string, cols, rows int) paneViewCacheKey {
	return paneViewCacheKey{
		PaneID: paneID,
		Cols:   cols,
		Rows:   rows,
	}
}

func (c *clientConn) paneViewCacheGet(key paneViewCacheKey) (PaneViewResponse, bool) {
	if c == nil {
		return PaneViewResponse{}, false
	}
	entry, ok := c.paneViewCacheGetEntry(key)
	if !ok {
		return PaneViewResponse{}, false
	}
	return entry.resp, true
}

func (c *clientConn) paneViewCacheGetEntry(key paneViewCacheKey) (cachedPaneView, bool) {
	if c == nil {
		return cachedPaneView{}, false
	}
	now := time.Now()

	c.paneViewCacheMu.Lock()
	if c.paneViewCache == nil {
		c.paneViewCacheMu.Unlock()
		return cachedPaneView{}, false
	}
	entry, ok := c.paneViewCache[key]
	if ok {
		if paneViewCacheTTL > 0 && now.Sub(entry.renderedAt) > paneViewCacheTTL {
			delete(c.paneViewCache, key)
			ok = false
		} else {
			entry.renderedAt = now
			c.paneViewCache[key] = entry
		}
	}
	c.paneViewCacheMu.Unlock()
	return entry, ok
}

func (c *clientConn) paneViewCachePut(key paneViewCacheKey, resp PaneViewResponse) {
	if c == nil {
		return
	}
	now := time.Now()
	c.paneViewCacheMu.Lock()
	if c.paneViewCache == nil {
		c.paneViewCache = make(map[paneViewCacheKey]cachedPaneView)
	}
	c.paneViewCache[key] = cachedPaneView{resp: resp, renderedAt: now, updateSeq: resp.UpdateSeq}
	c.paneViewCachePruneLocked(now)
	c.paneViewCacheMu.Unlock()
}

func (c *clientConn) paneViewCachePruneLocked(now time.Time) {
	if c == nil || c.paneViewCache == nil {
		return
	}
	if paneViewCacheTTL > 0 {
		for k, e := range c.paneViewCache {
			if now.Sub(e.renderedAt) > paneViewCacheTTL {
				delete(c.paneViewCache, k)
			}
		}
	}
	if paneViewCacheMaxEntries < 1 || len(c.paneViewCache) <= paneViewCacheMaxEntries {
		return
	}

	type evictItem struct {
		key paneViewCacheKey
		at  time.Time
	}
	items := make([]evictItem, 0, len(c.paneViewCache))
	for k, e := range c.paneViewCache {
		items = append(items, evictItem{key: k, at: e.renderedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].at.Before(items[j].at)
	})
	extra := len(items) - paneViewCacheMaxEntries
	for i := 0; i < extra; i++ {
		delete(c.paneViewCache, items[i].key)
	}
}

func (d *Daemon) startPaneViewWorkers(client *clientConn) {
	if client == nil {
		return
	}
	workers := paneViewMaxConcurrency
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		d.wg.Add(1)
		go d.paneViewWorker(client)
	}
}

func (d *Daemon) paneViewWorker(client *clientConn) {
	defer d.wg.Done()
	if client == nil || client.paneViews == nil {
		return
	}
	for {
		select {
		case <-client.done:
			return
		case <-d.ctx.Done():
			return
		default:
		}

		paneID, job, ok := client.paneViews.next()
		if !ok {
			return
		}
		resp, send := d.paneViewResponseEnvelope(client, job, paneID)
		client.paneViews.finish(paneID)
		if !send {
			continue
		}
		timeout := d.responseTimeout(job.env)
		if err := sendEnvelope(client, resp, timeout); err != nil {
			d.shutdownClientConn(client)
			return
		}
	}
}

func (d *Daemon) paneViewResponseEnvelope(client *clientConn, job paneViewJob, paneID string) (Envelope, bool) {
	resp := Envelope{Kind: EnvelopeResponse, Op: job.env.Op, ID: job.env.ID}
	ctx, cancel := d.paneViewContext(client, paneID, job.req)
	if cancel != nil {
		defer cancel()
	}
	if client != nil && client.paneViews != nil {
		defer client.paneViews.clearCancel(paneID)
	}
	perf := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	var computeStart time.Time
	if perf {
		computeStart = time.Now()
	}
	viewResp, err := d.paneViewResponse(ctx, client, paneID, job.req)
	if client != nil && client.paneViews != nil {
		if !client.paneViews.isLatest(paneID, job.seq) {
			return Envelope{}, false
		}
	}
	if err != nil {
		resp.Error = err.Error()
		return resp, true
	}
	if perf && !computeStart.IsZero() {
		d.logPaneViewFirstAfterOutput(paneID, job.received, computeStart, time.Now(), viewResp)
	}
	payload, err := encodePayload(viewResp)
	if err != nil {
		resp.Error = err.Error()
		return resp, true
	}
	resp.Payload = payload
	return resp, true
}

func (d *Daemon) paneViewContext(client *clientConn, paneID string, req PaneViewRequest) (context.Context, context.CancelFunc) {
	base := d.ctx
	if base == nil {
		base = context.Background()
	}
	deadline, ok := paneViewDeadline(req)
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if ok {
		ctx, cancel = context.WithDeadline(base, deadline)
	} else {
		ctx, cancel = context.WithCancel(base)
	}
	if client != nil && client.paneViews != nil {
		client.paneViews.setCancel(paneID, cancel)
	}
	return ctx, cancel
}

func (d *Daemon) paneViewResponse(ctx context.Context, client *clientConn, paneID string, req PaneViewRequest) (PaneViewResponse, error) {
	manager, err := d.requireManager()
	if err != nil {
		return PaneViewResponse{}, err
	}
	win := manager.Window(paneID)
	if win == nil {
		if d.restore != nil {
			if snap, ok := d.restore.Snapshot(paneID); ok {
				return offlinePaneView(req, snap), nil
			}
		}
		return PaneViewResponse{}, fmt.Errorf("sessiond: pane %q not found", paneID)
	}
	info := buildPaneViewRenderInfo(win, paneID, req)

	if err := ctx.Err(); err != nil {
		return PaneViewResponse{}, err
	}
	if err := win.Resize(info.cols, info.rows); err != nil {
		return PaneViewResponse{}, err
	}

	currentSeq, knownSeq := paneViewSeqs(win, req)
	if resp, ok := paneViewNotModified(win, paneID, info, currentSeq, knownSeq); ok {
		return resp, nil
	}
	if resp, ok := paneViewCachedHit(client, win, paneID, info, currentSeq); ok {
		return resp, nil
	}
	if resp, err, ok := paneViewDeadlineFallback(ctx, client, info.key); ok {
		return resp, err
	}

	var frame termframe.Frame
	perf := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	if perf {
		start := time.Now()
		frame, err = paneViewFrame(ctx, win, info.renderReq)
		dur := time.Since(start)
		if dur > paneViewSlowThreshold {
			logging.LogEvery(
				context.Background(),
				"sessiond.paneview.render",
				2*time.Second,
				slog.LevelDebug,
				"sessiond: pane view render slow",
				slog.String("pane_id", paneID),
				slog.Duration("dur", dur),
				slog.Int("cols", info.cols),
				slog.Int("rows", info.rows),
			)
		}
	} else {
		frame, err = paneViewFrame(ctx, win, info.renderReq)
	}
	if err != nil {
		return paneViewRenderError(err, client, info.key)
	}
	d.logPaneViewFirst(win, paneID, info.cols, info.rows, frame)

	// Refresh seq after render. If output happened concurrently, the next request will pick it up.
	currentSeq = paneViewEffectiveSeq(win)

	resp := PaneViewResponse{
		PaneID:      paneID,
		Cols:        info.cols,
		Rows:        info.rows,
		UpdateSeq:   currentSeq,
		NotModified: false,
		Frame:       frame,
		HasMouse:    win.HasMouseMode(),
		AllowMotion: win.AllowsMouseMotion(),
	}
	if client != nil {
		client.paneViewCachePut(info.key, resp)
	}
	return resp, nil
}

type paneViewRenderInfo struct {
	cols      int
	rows      int
	renderReq PaneViewRequest
	key       paneViewCacheKey
}

func buildPaneViewRenderInfo(win paneViewWindow, paneID string, req PaneViewRequest) paneViewRenderInfo {
	cols, rows := normalizeDimensions(req.Cols, req.Rows)
	renderReq := req
	renderReq.Cols = cols
	renderReq.Rows = rows
	return paneViewRenderInfo{
		cols:      cols,
		rows:      rows,
		renderReq: renderReq,
		key:       paneViewCacheKeyFor(paneID, cols, rows),
	}
}

func paneViewSeqs(win paneViewWindow, req PaneViewRequest) (uint64, uint64) {
	return paneViewEffectiveSeq(win), req.KnownSeq
}

func paneViewNotModified(
	win paneWindow,
	paneID string,
	info paneViewRenderInfo,
	currentSeq uint64,
	knownSeq uint64,
) (PaneViewResponse, bool) {
	if knownSeq == 0 || knownSeq != currentSeq {
		return PaneViewResponse{}, false
	}
	return PaneViewResponse{
		PaneID:      paneID,
		Cols:        info.cols,
		Rows:        info.rows,
		UpdateSeq:   currentSeq,
		NotModified: true,
		Frame:       termframe.Frame{},
		HasMouse:    win.HasMouseMode(),
		AllowMotion: win.AllowsMouseMotion(),
	}, true
}

func paneViewCachedHit(
	client *clientConn,
	win paneWindow,
	paneID string,
	info paneViewRenderInfo,
	currentSeq uint64,
) (PaneViewResponse, bool) {
	if client == nil {
		return PaneViewResponse{}, false
	}
	entry, ok := client.paneViewCacheGetEntry(info.key)
	if !ok || entry.updateSeq != currentSeq {
		return PaneViewResponse{}, false
	}
	cached := entry.resp
	cached.PaneID = paneID
	cached.Cols = info.cols
	cached.Rows = info.rows
	cached.UpdateSeq = currentSeq
	cached.NotModified = false
	cached.HasMouse = win.HasMouseMode()
	cached.AllowMotion = win.AllowsMouseMotion()
	return cached, true
}

func paneViewDeadlineFallback(ctx context.Context, client *clientConn, key paneViewCacheKey) (PaneViewResponse, error, bool) {
	if !paneViewDeadlineSoon(ctx) {
		return PaneViewResponse{}, nil, false
	}
	if client != nil {
		if cached, ok := client.paneViewCacheGet(key); ok {
			return cached, nil, true
		}
	}
	return PaneViewResponse{}, context.DeadlineExceeded, true
}

func paneViewRenderError(err error, client *clientConn, key paneViewCacheKey) (PaneViewResponse, error) {
	if client != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
		if cached, ok := client.paneViewCacheGet(key); ok {
			return cached, nil
		}
	}
	return PaneViewResponse{}, err
}

func (d *Daemon) handlePaneView(payload []byte) ([]byte, error) {
	var req PaneViewRequest
	if err := decodePayload(payload, &req); err != nil {
		return nil, err
	}
	paneID, err := requirePaneID(req.PaneID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := d.paneViewContext(nil, paneID, req)
	if cancel != nil {
		defer cancel()
	}
	resp, err := d.paneViewResponse(ctx, nil, paneID, req)
	if err != nil {
		return nil, err
	}
	return encodePayload(resp)
}

func paneViewDeadline(req PaneViewRequest) (time.Time, bool) {
	if req.DeadlineUnixNano <= 0 {
		return time.Time{}, false
	}
	return time.Unix(0, req.DeadlineUnixNano), true
}

func paneViewDeadlineSoon(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return false
	}
	return time.Until(deadline) <= paneViewDeadlineSlack
}

func paneViewFrame(ctx context.Context, win paneViewWindow, req PaneViewRequest) (termframe.Frame, error) {
	if req.DirectRender {
		return win.ViewFrameDirectCtx(ctx)
	}
	return win.ViewFrameCtx(ctx)
}
