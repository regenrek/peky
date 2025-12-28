package peakypanes

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

const paneViewTimeout = 2 * time.Second

type paneViewKey struct {
	PaneID     string
	Cols       int
	Rows       int
	Mode       sessiond.PaneViewMode
	ShowCursor bool
}

func paneViewKeyFrom(view sessiond.PaneViewResponse) paneViewKey {
	return paneViewKey{
		PaneID:     view.PaneID,
		Cols:       view.Cols,
		Rows:       view.Rows,
		Mode:       view.Mode,
		ShowCursor: view.ShowCursor,
	}
}

func (m *Model) refreshPaneViewsCmd() tea.Cmd {
	if m == nil {
		return nil
	}
	return m.fetchPaneViewsCmd(m.paneViewRequests())
}

func (m *Model) refreshPaneViewFor(paneID string) tea.Cmd {
	if m == nil || paneID == "" {
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
	return m.fetchPaneViewsCmd([]sessiond.PaneViewRequest{*req})
}

func (m *Model) paneHitFor(paneID string) (paneHit, bool) {
	for _, hit := range m.paneHits() {
		if hit.PaneID == paneID {
			return hit, true
		}
	}
	return paneHit{}, false
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
			PaneID:     req.PaneID,
			Cols:       req.Cols,
			Rows:       req.Rows,
			Mode:       req.Mode,
			ShowCursor: req.ShowCursor,
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		reqs = append(reqs, *req)
	}
	return reqs
}

func (m *Model) paneViewRequestForHit(hit paneHit) *sessiond.PaneViewRequest {
	if hit.PaneID == "" {
		return nil
	}
	if hit.Content.empty() {
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
		PaneID:     hit.PaneID,
		Cols:       cols,
		Rows:       rows,
		Mode:       mode,
		ShowCursor: showCursor,
	}
}

func (m *Model) fetchPaneViewsCmd(reqs []sessiond.PaneViewRequest) tea.Cmd {
	if m == nil || m.client == nil || len(reqs) == 0 {
		return nil
	}
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), paneViewTimeout)
		defer cancel()
		views := make([]sessiond.PaneViewResponse, 0, len(reqs))
		var firstErr error
		for _, req := range reqs {
			resp, err := client.GetPaneView(ctx, req)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			views = append(views, resp)
		}
		if len(views) == 0 && firstErr != nil {
			return paneViewsMsg{Err: firstErr}
		}
		return paneViewsMsg{Views: views, Err: firstErr}
	}
}

func (m Model) paneView(paneID string, cols, rows int, mode sessiond.PaneViewMode, showCursor bool) string {
	if paneID == "" || cols <= 0 || rows <= 0 {
		return ""
	}
	key := paneViewKey{
		PaneID:     paneID,
		Cols:       cols,
		Rows:       rows,
		Mode:       mode,
		ShowCursor: showCursor,
	}
	if m.paneViews == nil {
		return ""
	}
	return m.paneViews[key]
}
