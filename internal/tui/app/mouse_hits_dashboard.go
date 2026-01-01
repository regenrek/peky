package app

import (
	"fmt"

	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func (m *Model) dashboardPaneHits() []mouse.PaneHit {
	body, ok := m.dashboardBodyRect()
	if !ok {
		m.logPaneViewSkipGlobal("body_rect_unavailable", m.paneViewSkipContext())
		if perfDebugEnabled() {
			logPerfEvery("tui.panehits.dashboard.body", perfLogInterval,
				"tui: pane hits dashboard skip reason=body_rect_unavailable %s", m.paneViewSkipContext())
		}
		return nil
	}
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		m.logPaneViewSkipGlobal("no_columns", m.dashboardColumnsDebug())
		if perfDebugEnabled() {
			logPerfEvery("tui.panehits.dashboard.columns", perfLogInterval,
				"tui: pane hits dashboard skip reason=no_columns %s", m.dashboardColumnsDebug())
		}
		return nil
	}
	columns = m.filteredDashboardColumns(columns)
	if countDashboardPanes(columns) == 0 {
		m.logPaneViewSkipGlobal("no_panes", m.paneViewSkipContext())
		if perfDebugEnabled() {
			logPerfEvery("tui.panehits.dashboard.nopanes", perfLogInterval,
				"tui: pane hits dashboard skip reason=no_panes %s", m.paneViewSkipContext())
		}
		return nil
	}

	selectedProject := dashboardSelectedProject(columns, m.selection)
	selectedIndex := dashboardColumnIndex(columns, selectedProject)
	layout := buildDashboardHitLayout(body, columns, selectedIndex, m.settings)
	if len(layout.columns) == 0 || layout.bodyHeight <= 0 {
		m.logPaneViewSkipGlobal("layout_empty", m.paneViewSkipContext())
		if perfDebugEnabled() {
			logPerfEvery("tui.panehits.dashboard.layout", perfLogInterval,
				"tui: pane hits dashboard skip reason=layout_empty columns=%d body_h=%d %s", len(layout.columns), layout.bodyHeight, m.paneViewSkipContext())
		}
		return nil
	}

	hits := make([]mouse.PaneHit, 0, countDashboardPanes(layout.columns))
	for i, column := range layout.columns {
		hits = append(hits, m.dashboardColumnHits(body, layout, column, i, m.selection)...)
	}
	return hits
}

type dashboardHitLayout struct {
	columns       []DashboardProjectColumn
	selectedIndex int
	gap           int
	colWidth      int
	headerHeight  int
	bodyHeight    int
	blockHeight   int
	visibleBlocks int
	previewLines  int
}

func countDashboardPanes(columns []DashboardProjectColumn) int {
	total := 0
	for _, column := range columns {
		total += len(column.Panes)
	}
	return total
}

func dashboardColumnIndex(columns []DashboardProjectColumn, selectedProject string) int {
	for i, column := range columns {
		if column.ProjectID == selectedProject {
			return i
		}
	}
	return 0
}

func buildDashboardHitLayout(body mouse.Rect, columns []DashboardProjectColumn, selectedIndex int, settings DashboardConfig) dashboardHitLayout {
	layout := dashboardHitLayout{
		columns:      columns,
		gap:          2,
		headerHeight: 3,
		previewLines: dashboardPreviewLines(settings),
	}
	layout.columns, layout.selectedIndex = clampDashboardColumns(columns, selectedIndex, body.W, layout.gap)
	layout.colWidth = columnWidth(body.W, len(layout.columns), layout.gap)
	layout.bodyHeight = body.H - layout.headerHeight
	layout.blockHeight = paneBlockHeight(layout.previewLines, layout.bodyHeight)
	layout.visibleBlocks = visibleBlockCount(layout.bodyHeight, layout.blockHeight)
	return layout
}

func clampDashboardColumns(columns []DashboardProjectColumn, selectedIndex, bodyWidth, gap int) ([]DashboardProjectColumn, int) {
	minColWidth := 24
	maxCols := (bodyWidth + gap) / (minColWidth + gap)
	if maxCols < 1 {
		maxCols = 1
	}
	if len(columns) <= maxCols {
		return columns, selectedIndex
	}
	start := selectedIndex - maxCols/2
	if start < 0 {
		start = 0
	}
	if start+maxCols > len(columns) {
		start = len(columns) - maxCols
	}
	return columns[start : start+maxCols], selectedIndex - start
}

func columnWidth(bodyWidth, columns, gap int) int {
	if columns <= 1 {
		if bodyWidth < 1 {
			return 1
		}
		return bodyWidth
	}
	width := (bodyWidth - gap*(columns-1)) / columns
	if width < 1 {
		width = 1
	}
	return width
}

func paneBlockHeight(previewLines, bodyHeight int) int {
	if bodyHeight <= 0 {
		return 0
	}
	blockHeight := dashboardPaneBlockHeight(previewLines)
	if blockHeight > bodyHeight {
		blockHeight = bodyHeight
	}
	if blockHeight < 3 {
		blockHeight = bodyHeight
	}
	return blockHeight
}

func visibleBlockCount(bodyHeight, blockHeight int) int {
	if bodyHeight <= 0 || blockHeight <= 0 {
		return 0
	}
	visible := bodyHeight / blockHeight
	if visible < 1 {
		visible = 1
	}
	return visible
}

func (m *Model) dashboardColumnHits(body mouse.Rect, layout dashboardHitLayout, column DashboardProjectColumn, index int, selection selectionState) []mouse.PaneHit {
	if len(column.Panes) == 0 || layout.bodyHeight <= 0 || layout.blockHeight <= 0 {
		return nil
	}

	selectedPaneIndex := -1
	if index == layout.selectedIndex {
		selectedPaneIndex = dashboardPaneIndex(column.Panes, selection)
	}
	forceAll := perfPaneViewAllEnabled()
	if forceAll && len(column.Panes) > 0 {
		if layout.visibleBlocks > 0 && len(column.Panes) > layout.visibleBlocks {
			m.logPaneViewSkipGlobal("perf_all_panes", fmt.Sprintf("project=%s column=%d total=%d visible=%d selected_idx=%d", column.ProjectID, index, len(column.Panes), layout.visibleBlocks, selectedPaneIndex))
		}
		return dashboardColumnHitsRange(body, layout, column, index, 0, len(column.Panes))
	}
	start, end, trimmed := dashboardPaneRange(index == layout.selectedIndex, selectedPaneIndex, layout.visibleBlocks, len(column.Panes))
	if trimmed {
		m.logPaneViewSkipGlobal("range_trimmed", fmt.Sprintf("project=%s column=%d total=%d visible=%d start=%d end=%d selected_idx=%d", column.ProjectID, index, len(column.Panes), layout.visibleBlocks, start, end, selectedPaneIndex))
	}
	return dashboardColumnHitsRange(body, layout, column, index, start, end)
}

func dashboardColumnHitsRange(body mouse.Rect, layout dashboardHitLayout, column DashboardProjectColumn, index int, start, end int) []mouse.PaneHit {
	if end <= start {
		return nil
	}
	colX := body.X + index*(layout.colWidth+layout.gap)
	hits := make([]mouse.PaneHit, 0, end-start)
	for idx := start; idx < end; idx++ {
		pane := column.Panes[idx]
		outer := mouse.Rect{
			X: colX,
			Y: body.Y + layout.headerHeight + (idx-start)*layout.blockHeight,
			W: layout.colWidth,
			H: layout.blockHeight,
		}
		content := dashboardPaneContentRect(outer, layout.previewLines)
		hits = append(hits, mouse.PaneHit{
			PaneID: pane.Pane.ID,
			Selection: mouse.Selection{
				ProjectID: column.ProjectID,
				Session:   pane.SessionName,
				Pane:      pane.Pane.Index,
			},
			Outer:   outer,
			Content: content,
		})
	}
	return hits
}

func dashboardPaneRange(isSelectedColumn bool, selectedPaneIndex, visibleBlocks, totalPanes int) (int, int, bool) {
	start := 0
	trimmed := false
	if isSelectedColumn && selectedPaneIndex >= 0 && selectedPaneIndex >= visibleBlocks {
		start = selectedPaneIndex - visibleBlocks + 1
	}
	if start < 0 {
		start = 0
	}
	if totalPanes > 0 && start > totalPanes-1 {
		start = totalPanes - 1
	}
	end := start + visibleBlocks
	if end > totalPanes {
		end = totalPanes
	}
	if end < start {
		end = start
	}
	if totalPanes > visibleBlocks {
		trimmed = true
	}
	return start, end, trimmed
}
