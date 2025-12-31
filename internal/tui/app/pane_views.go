package app

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/diag"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

const (
	paneViewTimeout               = 2 * time.Second
	paneViewMaxConcurrency        = 4
	paneViewMaxBatch              = 8
	paneViewMinIntervalFocused    = 33 * time.Millisecond
	paneViewMinIntervalSelected   = 100 * time.Millisecond
	paneViewMinIntervalBackground = 250 * time.Millisecond
	paneViewTimeoutFocused        = 1500 * time.Millisecond
	paneViewTimeoutSelected       = 1000 * time.Millisecond
	paneViewTimeoutBackground     = 800 * time.Millisecond
)

type paneViewKey struct {
	PaneID       string
	Cols         int
	Rows         int
	Mode         sessiond.PaneViewMode
	ShowCursor   bool
	ColorProfile termenv.Profile
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
	if m.paneViewInFlight > 0 {
		m.paneViewQueued = true
		diag.LogEvery("tui.paneviews.queue", 2*time.Second, "tui: pane views queued in_flight=%d", m.paneViewInFlight)
		return nil
	}
	return m.startPaneViewFetch(m.paneViewRequests())
}

func (m *Model) refreshPaneViewFor(paneID string) tea.Cmd {
	if m == nil || paneID == "" {
		return nil
	}
	if m.paneViewInFlight > 0 {
		m.queuePaneViewID(paneID)
		return nil
	}
	hit, ok := m.paneHitFor(paneID)
	if !ok {
		return nil
	}
	req := m.paneViewRequestForHit(hit)
	if req == nil {
		return nil
	}

	key := paneViewKey{
		PaneID:       req.PaneID,
		Cols:         req.Cols,
		Rows:         req.Rows,
		Mode:         req.Mode,
		ShowCursor:   req.ShowCursor,
		ColorProfile: req.ColorProfile,
	}
	if !m.allowPaneViewRequest(key, paneViewMinIntervalFor(*req)) {
		return nil
	}

	return m.startPaneViewFetch([]sessiond.PaneViewRequest{*req})
}

func (m *Model) refreshPaneViewsForIDs(ids map[string]struct{}) tea.Cmd {
	if m == nil || len(ids) == 0 {
		return nil
	}
	if m.paneViewInFlight > 0 {
		for id := range ids {
			m.queuePaneViewID(id)
		}
		return nil
	}
	return m.startPaneViewFetch(m.paneViewRequestsForIDs(ids))
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
	hits := m.paneHits()
	if len(hits) == 0 {
		return nil
	}
	reqs := make([]sessiond.PaneViewRequest, 0, len(hits))
	seen := make(map[paneViewKey]struct{})
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

		if !m.allowPaneViewRequest(key, paneViewMinIntervalFor(*req)) {
			continue
		}

		reqs = append(reqs, *req)
	}
	return reqs
}

func (m *Model) startPaneViewFetch(reqs []sessiond.PaneViewRequest) tea.Cmd {
	if m == nil || m.client == nil || len(reqs) == 0 {
		return nil
	}
	chunks := chunkPaneViewRequests(reqs, paneViewMaxBatch)
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

func (m *Model) queuePaneViewID(paneID string) {
	if m == nil || paneID == "" {
		return
	}
	if m.paneViewQueuedIDs == nil {
		m.paneViewQueuedIDs = make(map[string]struct{})
	}
	m.paneViewQueuedIDs[paneID] = struct{}{}
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

func (m *Model) paneViewRequestsForIDs(ids map[string]struct{}) []sessiond.PaneViewRequest {
	if m == nil || len(ids) == 0 {
		return nil
	}
	reqs := make([]sessiond.PaneViewRequest, 0, len(ids))
	seen := make(map[paneViewKey]struct{})
	for paneID := range ids {
		hit, ok := m.paneHitFor(paneID)
		if !ok {
			continue
		}
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
		reqs = append(reqs, *req)
	}
	return reqs
}

func (m *Model) paneViewRequestForHit(hit mouse.PaneHit) *sessiond.PaneViewRequest {
	if hit.PaneID == "" {
		return nil
	}
	if hit.Content.Empty() {
		return nil
	}
	cols := hit.Content.W
	rows := hit.Content.H
	if cols <= 0 || rows <= 0 {
		return nil
	}

	// Active-pane Lipgloss only:
	// - Focused pane (selected + terminal focus) gets Lipgloss for cursor overlay.
	// - Everything else uses ANSI for speed and fidelity.
	mode := sessiond.PaneViewANSI
	showCursor := false

	selectedID := ""
	if pane := m.selectedPane(); pane != nil {
		selectedID = pane.ID
	}
	isSelected := selectedID != "" && selectedID == hit.PaneID

	priority := sessiond.PaneViewPriorityBackground
	if isSelected {
		priority = sessiond.PaneViewPriorityNormal
	}

	if isSelected && m.terminalFocus && m.supportsTerminalFocus() {
		mode = sessiond.PaneViewLipgloss
		showCursor = true
		priority = sessiond.PaneViewPriorityFocused
	}

	req := &sessiond.PaneViewRequest{
		PaneID:       hit.PaneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         mode,
		ShowCursor:   showCursor,
		ColorProfile: m.paneViewProfile,

		Priority: priority,
	}

	key := paneViewKey{
		PaneID:       req.PaneID,
		Cols:         req.Cols,
		Rows:         req.Rows,
		Mode:         req.Mode,
		ShowCursor:   req.ShowCursor,
		ColorProfile: req.ColorProfile,
	}
	if m.paneViewSeq != nil {
		req.KnownSeq = m.paneViewSeq[key]
	}

	return req
}

func (m *Model) allowPaneViewRequest(key paneViewKey, minInterval time.Duration) bool {
	if m == nil || key.PaneID == "" {
		return false
	}
	if m.paneViewLastReq == nil {
		m.paneViewLastReq = make(map[paneViewKey]time.Time)
	}
	now := time.Now()
	if last, ok := m.paneViewLastReq[key]; ok {
		if minInterval > 0 && now.Sub(last) < minInterval {
			diag.LogEvery("tui.paneviews.throttle", 2*time.Second, "tui: pane view throttled pane=%s interval=%s", key.PaneID, minInterval)
			return false
		}
	}
	m.paneViewLastReq[key] = now
	return true
}

func paneViewMinIntervalFor(req sessiond.PaneViewRequest) time.Duration {
	if req.Priority == sessiond.PaneViewPriorityFocused || req.Mode == sessiond.PaneViewLipgloss || req.ShowCursor {
		return paneViewMinIntervalFocused
	}
	if req.Priority == sessiond.PaneViewPriorityNormal || req.Priority == sessiond.PaneViewPriorityUnset {
		return paneViewMinIntervalSelected
	}
	return paneViewMinIntervalBackground
}

func paneViewTimeoutFor(req sessiond.PaneViewRequest) time.Duration {
	if req.Priority == sessiond.PaneViewPriorityFocused || req.Mode == sessiond.PaneViewLipgloss || req.ShowCursor {
		return paneViewTimeoutFocused
	}
	if req.Priority == sessiond.PaneViewPriorityNormal || req.Priority == sessiond.PaneViewPriorityUnset {
		return paneViewTimeoutSelected
	}
	return paneViewTimeoutBackground
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
	return func() tea.Msg {
		return fetchPaneViews(client, reqs)
	}
}

type paneViewResult struct {
	view sessiond.PaneViewResponse
	err  error
}

func fetchPaneViews(client *sessiond.Client, reqs []sessiond.PaneViewRequest) paneViewsMsg {
	start := time.Now()
	results := make(chan paneViewResult, len(reqs))
	jobs := make(chan sessiond.PaneViewRequest)

	workers := paneViewWorkerCount(reqs)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go paneViewWorker(client, jobs, results, &wg)
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
		return paneViewsMsg{Err: firstErr}
	}
	return paneViewsMsg{Views: views, Err: firstErr}
}

func paneViewWorkerCount(reqs []sessiond.PaneViewRequest) int {
	workers := paneViewMaxConcurrency
	if len(reqs) < workers {
		workers = len(reqs)
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}

func paneViewWorker(client *sessiond.Client, jobs <-chan sessiond.PaneViewRequest, results chan<- paneViewResult, wg *sync.WaitGroup) {
	defer wg.Done()
	for req := range jobs {
		timeout := paneViewTimeoutFor(req)
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
	return m.paneViews[key]
}

func detectPaneViewProfile() termenv.Profile {
	return termenv.EnvColorProfile()
}
