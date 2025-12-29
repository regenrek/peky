package app

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

const (
	paneViewTimeout        = 2 * time.Second
	paneViewMaxConcurrency = 4
	paneViewMinInterval    = 50 * time.Millisecond
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
	if m.paneViewInFlight {
		m.paneViewQueued = true
		return nil
	}
	return m.startPaneViewFetch(m.paneViewRequests())
}

func (m *Model) refreshPaneViewFor(paneID string) tea.Cmd {
	if m == nil || paneID == "" {
		return nil
	}
	if m.paneViewInFlight {
		m.queuePaneViewID(paneID)
		return nil
	}
	if !m.allowPaneViewRequest(paneID) {
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
	return m.startPaneViewFetch([]sessiond.PaneViewRequest{*req})
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
		reqs = append(reqs, *req)
	}
	return reqs
}

func (m *Model) startPaneViewFetch(reqs []sessiond.PaneViewRequest) tea.Cmd {
	if m == nil || m.client == nil || len(reqs) == 0 {
		return nil
	}
	m.paneViewInFlight = true
	return m.fetchPaneViewsCmd(reqs)
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
	mode := sessiond.PaneViewANSI
	showCursor := false
	if m.terminalFocus && m.supportsTerminalFocus() {
		if pane := m.selectedPane(); pane != nil && pane.ID == hit.PaneID {
			mode = sessiond.PaneViewLipgloss
			showCursor = true
		}
	}
	return &sessiond.PaneViewRequest{
		PaneID:       hit.PaneID,
		Cols:         cols,
		Rows:         rows,
		Mode:         mode,
		ShowCursor:   showCursor,
		ColorProfile: m.paneViewProfile,
	}
}

func (m *Model) allowPaneViewRequest(paneID string) bool {
	if m == nil || paneID == "" {
		return false
	}
	if m.paneViewLastReq == nil {
		m.paneViewLastReq = make(map[string]time.Time)
	}
	now := time.Now()
	if last, ok := m.paneViewLastReq[paneID]; ok {
		if now.Sub(last) < paneViewMinInterval {
			return false
		}
	}
	m.paneViewLastReq[paneID] = now
	return true
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
		views := make([]sessiond.PaneViewResponse, 0, len(reqs))
		var firstErr error

		type result struct {
			view sessiond.PaneViewResponse
			err  error
		}
		results := make(chan result, len(reqs))
		jobs := make(chan sessiond.PaneViewRequest)

		workers := paneViewMaxConcurrency
		if len(reqs) < workers {
			workers = len(reqs)
		}
		if workers < 1 {
			workers = 1
		}

		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				for req := range jobs {
					ctx, cancel := context.WithTimeout(context.Background(), paneViewTimeout)
					if deadline, ok := ctx.Deadline(); ok {
						req.DeadlineUnixNano = deadline.UnixNano()
					}
					resp, err := client.GetPaneView(ctx, req)
					cancel()
					if err != nil {
						results <- result{err: err}
						continue
					}
					results <- result{view: resp}
				}
			}()
		}

		go func() {
			for _, req := range reqs {
				jobs <- req
			}
			close(jobs)
			wg.Wait()
			close(results)
		}()

		for res := range results {
			if res.err != nil {
				if firstErr == nil {
					firstErr = res.err
				}
				continue
			}
			views = append(views, res.view)
		}

		if len(views) == 0 && firstErr != nil {
			return paneViewsMsg{Err: firstErr}
		}
		return paneViewsMsg{Views: views, Err: firstErr}
	}
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
