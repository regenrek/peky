package sessiond

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/muesli/termenv"
)

const (
	paneViewMaxConcurrency = 4
	paneViewDeadlineSlack  = 50 * time.Millisecond
)

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
	next := s.seq[paneID] + 1
	s.seq[paneID] = next
	s.pending[paneID] = paneViewJob{env: env, req: req, seq: next, received: time.Now()}
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
	for paneID, job := range s.pending {
		if s.inflight[paneID] {
			continue
		}
		return paneID, job, true
	}
	return "", paneViewJob{}, false
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
	PaneID       string
	Cols         int
	Rows         int
	Mode         PaneViewMode
	ShowCursor   bool
	ColorProfile termenv.Profile
}

type cachedPaneView struct {
	resp       PaneViewResponse
	renderedAt time.Time
}

func paneViewCacheKeyFor(paneID string, cols, rows int, req PaneViewRequest) paneViewCacheKey {
	return paneViewCacheKey{
		PaneID:       paneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         req.Mode,
		ShowCursor:   req.ShowCursor,
		ColorProfile: req.ColorProfile,
	}
}

func (c *clientConn) paneViewCacheGet(key paneViewCacheKey) (PaneViewResponse, bool) {
	if c == nil {
		return PaneViewResponse{}, false
	}
	c.paneViewCacheMu.Lock()
	entry, ok := c.paneViewCache[key]
	c.paneViewCacheMu.Unlock()
	if !ok {
		return PaneViewResponse{}, false
	}
	return entry.resp, true
}

func (c *clientConn) paneViewCachePut(key paneViewCacheKey, resp PaneViewResponse) {
	if c == nil {
		return
	}
	c.paneViewCacheMu.Lock()
	if c.paneViewCache == nil {
		c.paneViewCache = make(map[paneViewCacheKey]cachedPaneView)
	}
	c.paneViewCache[key] = cachedPaneView{resp: resp, renderedAt: time.Now()}
	c.paneViewCacheMu.Unlock()
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
		return PaneViewResponse{}, fmt.Errorf("sessiond: pane %q not found", paneID)
	}
	cols, rows := normalizeDimensions(req.Cols, req.Rows)
	key := paneViewCacheKeyFor(paneID, cols, rows, req)
	if err := ctx.Err(); err != nil {
		return PaneViewResponse{}, err
	}
	if err := win.Resize(cols, rows); err != nil {
		return PaneViewResponse{}, err
	}
	if paneViewDeadlineSoon(ctx) {
		if client != nil {
			if cached, ok := client.paneViewCacheGet(key); ok {
				return cached, nil
			}
		}
		return PaneViewResponse{}, context.DeadlineExceeded
	}
	view, err := paneViewString(ctx, win, req)
	if err != nil {
		if client != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
			if cached, ok := client.paneViewCacheGet(key); ok {
				return cached, nil
			}
		}
		return PaneViewResponse{}, err
	}
	resp := PaneViewResponse{
		PaneID:       paneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         req.Mode,
		ShowCursor:   req.ShowCursor,
		ColorProfile: req.ColorProfile,
		View:         view,
		HasMouse:     win.HasMouseMode(),
		AllowMotion:  win.AllowsMouseMotion(),
	}
	if client != nil {
		client.paneViewCachePut(key, resp)
	}
	return resp, nil
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

func paneViewString(ctx context.Context, win paneViewWindow, req PaneViewRequest) (string, error) {
	switch req.Mode {
	case PaneViewLipgloss:
		return win.ViewLipglossCtx(ctx, req.ShowCursor, req.ColorProfile)
	default:
		return win.ViewANSICtx(ctx)
	}
}
