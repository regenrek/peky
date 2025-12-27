package peakypanes

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tmuxstream"
)

func (m *Model) syncTmuxStreamsCmd() tea.Cmd {
	if m.tmuxStreams == nil {
		return nil
	}
	specs := m.visibleTmuxPaneSpecs()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.tmuxStreams.SetDesired(ctx, specs); err != nil {
			return tmuxStreamSyncMsg{Err: err}
		}
		return nil
	}
}

func (m *Model) visibleTmuxPaneSpecs() []tmuxstream.PaneSpec {
	if m.state != StateDashboard {
		return nil
	}
	bodyW, bodyH := m.dashboardBodySize()
	if bodyW <= 0 || bodyH <= 0 {
		return nil
	}
	if m.tab == TabProject {
		return m.visibleProjectTmuxPanes()
	}
	return m.visibleDashboardTmuxPanes(bodyW, bodyH)
}

func (m *Model) visibleProjectTmuxPanes() []tmuxstream.PaneSpec {
	session := m.selectedSession()
	if session == nil || session.Status == StatusStopped {
		return nil
	}
	if session.Multiplexer != layout.MultiplexerTmux {
		return nil
	}
	winIndex := strings.TrimSpace(m.selection.Window)
	if winIndex == "" {
		winIndex = strings.TrimSpace(session.ActiveWindow)
	}
	win := selectedWindow(session, winIndex)
	if win == nil {
		return nil
	}

	out := make([]tmuxstream.PaneSpec, 0, len(win.Panes))
	for _, p := range win.Panes {
		if p.Multiplexer != layout.MultiplexerTmux {
			continue
		}
		if strings.TrimSpace(p.ID) == "" {
			continue
		}
		out = append(out, tmuxstream.PaneSpec{
			PaneID: p.ID,
			Cols:   p.Width,
			Rows:   p.Height,
		})
	}
	return out
}

func (m *Model) visibleDashboardTmuxPanes(width, height int) []tmuxstream.PaneSpec {
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		return nil
	}
	columns = m.filteredDashboardColumns(columns)
	selectedProject := m.dashboardSelectedProject(columns)
	previewLines := dashboardPreviewLines(m.settings)

	gap := 2
	minColWidth := 24
	maxCols := (width + gap) / (minColWidth + gap)
	if maxCols < 1 {
		maxCols = 1
	}

	selectedIndex := 0
	for i, c := range columns {
		if c.ProjectName == selectedProject {
			selectedIndex = i
			break
		}
	}

	if len(columns) > maxCols {
		start := selectedIndex - maxCols/2
		if start < 0 {
			start = 0
		}
		if start+maxCols > len(columns) {
			start = len(columns) - maxCols
		}
		columns = columns[start : start+maxCols]
		selectedIndex = selectedIndex - start
	}

	headerHeight := 3
	bodyHeight := height - headerHeight
	if bodyHeight <= 0 {
		return nil
	}

	blockHeight := dashboardPaneBlockHeight(previewLines)
	if blockHeight > bodyHeight {
		blockHeight = bodyHeight
	}
	if blockHeight < 3 {
		blockHeight = bodyHeight
	}
	visibleBlocks := bodyHeight / blockHeight
	if visibleBlocks < 1 {
		visibleBlocks = 1
	}

	var specs []tmuxstream.PaneSpec
	for ci, col := range columns {
		if len(col.Panes) == 0 {
			continue
		}
		selectedCol := ci == selectedIndex
		selectedPaneIndex := -1
		if selectedCol {
			selectedPaneIndex = dashboardPaneIndex(col.Panes, m.selection)
		}
		start := 0
		if selectedCol && selectedPaneIndex >= 0 && selectedPaneIndex >= visibleBlocks {
			start = selectedPaneIndex - visibleBlocks + 1
		}
		if start < 0 {
			start = 0
		}
		if start > len(col.Panes)-1 {
			start = len(col.Panes) - 1
		}
		end := start + visibleBlocks
		if end > len(col.Panes) {
			end = len(col.Panes)
		}

		for i := start; i < end; i++ {
			p := col.Panes[i].Pane
			if p.Multiplexer != layout.MultiplexerTmux {
				continue
			}
			if strings.TrimSpace(p.ID) == "" {
				continue
			}
			specs = append(specs, tmuxstream.PaneSpec{
				PaneID: p.ID,
				Cols:   p.Width,
				Rows:   p.Height,
			})
		}
	}

	return specs
}

func (m *Model) dashboardBodySize() (int, int) {
	if m.width == 0 || m.height == 0 {
		return 0, 0
	}
	h, v := appStyle.GetFrameSize()
	contentWidth := m.width - h
	contentHeight := m.height - v
	if contentWidth <= 10 || contentHeight <= 6 {
		return 0, 0
	}

	showThumbs := m.settings.ShowThumbnails
	thumbHeight := 0
	if showThumbs {
		thumbHeight = 3
	}

	header := m.viewHeader(contentWidth)
	footer := m.viewFooter(contentWidth)
	quickReply := m.viewQuickReply(contentWidth)
	quickReplyHeight := lipgloss.Height(quickReply)

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	headerGap := 1
	extraLines := headerHeight + headerGap + footerHeight + quickReplyHeight
	if showThumbs {
		extraLines += thumbHeight
	}
	bodyHeight := contentHeight - extraLines
	if bodyHeight < 4 {
		showThumbs = false
		thumbHeight = 0
		headerGap = 0
		extraLines = headerHeight + headerGap + footerHeight + quickReplyHeight
		bodyHeight = contentHeight - extraLines
	}

	return contentWidth, bodyHeight
}
