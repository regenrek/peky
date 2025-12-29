package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/icons"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

type paneIconContext struct {
	set  icons.IconSet
	size icons.Size
}

type dashboardPaneRenderer func(pane DashboardPane, width, height, previewLines int, selected bool, focused bool, iconCtx paneIconContext) string

type dashboardRenderContext struct {
	selectionSession string
	selectionPane    string
	previewLines     int
	terminalFocus    bool
	icons            paneIconContext
	renderer         dashboardPaneRenderer
}

func (ctx dashboardRenderContext) withIcons(iconSet icons.IconSet, iconSize icons.Size) dashboardRenderContext {
	ctx.icons = paneIconContext{set: iconSet, size: iconSize}
	return ctx
}

func renderDashboardColumnsWithRenderer(columns []DashboardColumn, width, height int, selectedProject string, ctx dashboardRenderContext) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(columns) == 0 {
		return padLines("No projects", width, height)
	}
	layout := buildDashboardColumnsLayout(columns, width, selectedProject)
	columns = layout.columns

	ctx = ctx.withIcons(icons.Active(), icons.ActiveSize())
	parts := make([]string, 0, len(columns)*2)
	for i, column := range columns {
		if i > 0 {
			parts = append(parts, strings.Repeat(" ", layout.gap))
		}
		selected := i == layout.selectedIndex
		parts = append(parts, renderDashboardColumn(column, layout.colWidth, height, selected, ctx))
	}
	return padLines(lipgloss.JoinHorizontal(lipgloss.Top, parts...), width, height)
}

func renderDashboardColumn(column DashboardColumn, width, height int, selected bool, ctx dashboardRenderContext) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	headerLines := dashboardColumnHeader(column, width, selected)
	bodyHeight := height - len(headerLines)
	if bodyHeight <= 0 {
		return padLines(strings.Join(headerLines, "\n"), width, height)
	}

	if len(column.Panes) == 0 {
		body := padLines("No running panes", width, bodyHeight)
		return strings.Join(append(headerLines, body), "\n")
	}

	layout := buildDashboardColumnLayout(column, bodyHeight, selected, ctx)
	blocks := renderDashboardPaneBlocks(column, layout, width, selected, ctx)
	body := padLines(strings.Join(blocks, "\n"), width, bodyHeight)
	return strings.Join(append(headerLines, body), "\n")
}

func dashboardPaneBlockHeight(previewLines int) int {
	if previewLines < 0 {
		previewLines = 0
	}
	return previewLines + 4
}

type dashboardColumnsLayout struct {
	columns       []DashboardColumn
	selectedIndex int
	gap           int
	colWidth      int
}

type dashboardColumnLayout struct {
	blockHeight   int
	visibleBlocks int
	start         int
	end           int
	selectedIndex int
}

func buildDashboardColumnsLayout(columns []DashboardColumn, width int, selectedProject string) dashboardColumnsLayout {
	layout := dashboardColumnsLayout{
		columns: columns,
		gap:     2,
	}
	layout.selectedIndex = dashboardColumnIndex(columns, selectedProject)
	layout.columns, layout.selectedIndex = clampDashboardColumns(layout.columns, layout.selectedIndex, width, layout.gap)
	layout.colWidth = columnWidth(width, len(layout.columns), layout.gap)
	return layout
}

func dashboardColumnIndex(columns []DashboardColumn, selectedProject string) int {
	for i, column := range columns {
		if column.ProjectID == selectedProject {
			return i
		}
	}
	return 0
}

func clampDashboardColumns(columns []DashboardColumn, selectedIndex, width, gap int) ([]DashboardColumn, int) {
	minColWidth := 24
	maxCols := (width + gap) / (minColWidth + gap)
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

func columnWidth(width, columns, gap int) int {
	if columns <= 1 {
		if width < 1 {
			return 1
		}
		return width
	}
	colWidth := (width - gap*(columns-1)) / columns
	if colWidth < 1 {
		colWidth = 1
	}
	return colWidth
}

func dashboardColumnHeader(column DashboardColumn, width int, selected bool) []string {
	name := strings.TrimSpace(column.ProjectName)
	if name == "" {
		name = "project"
	}
	path := pathOrDash(column.ProjectPath)
	headerStyle := theme.TabInactive
	if selected {
		headerStyle = theme.TabActive
	}
	return []string{
		fitLine(headerStyle.Render(name), width),
		fitLine(theme.SidebarMeta.Render(path), width),
		strings.Repeat("─", width),
	}
}

func buildDashboardColumnLayout(column DashboardColumn, bodyHeight int, selected bool, ctx dashboardRenderContext) dashboardColumnLayout {
	layout := dashboardColumnLayout{
		blockHeight: dashboardPaneBlockHeight(ctx.previewLines),
	}
	if layout.blockHeight > bodyHeight {
		layout.blockHeight = bodyHeight
	}
	if layout.blockHeight < 3 {
		layout.blockHeight = bodyHeight
	}
	layout.visibleBlocks = bodyHeight / layout.blockHeight
	if layout.visibleBlocks < 1 {
		layout.visibleBlocks = 1
	}
	if selected {
		layout.selectedIndex = dashboardPaneIndex(column.Panes, ctx.selectionSession, ctx.selectionPane)
	} else {
		layout.selectedIndex = -1
	}
	layout.start, layout.end = dashboardPaneRange(selected, layout.selectedIndex, layout.visibleBlocks, len(column.Panes))
	return layout
}

func renderDashboardPaneBlocks(column DashboardColumn, layout dashboardColumnLayout, width int, selected bool, ctx dashboardRenderContext) []string {
	if ctx.renderer == nil {
		ctx.renderer = renderDashboardPaneTile
	}
	blocks := make([]string, 0, layout.visibleBlocks)
	for i := layout.start; i < layout.end; i++ {
		selectedPane := selected && i == layout.selectedIndex
		focused := selectedPane && ctx.terminalFocus
		blocks = append(blocks, ctx.renderer(column.Panes[i], width, layout.blockHeight, ctx.previewLines, selectedPane, focused, ctx.icons))
	}
	return blocks
}

func dashboardPaneRange(selected bool, selectedIndex, visibleBlocks, total int) (int, int) {
	start := 0
	if selected && selectedIndex >= 0 && selectedIndex >= visibleBlocks {
		start = selectedIndex - visibleBlocks + 1
	}
	if start < 0 {
		start = 0
	}
	if total > 0 && start > total-1 {
		start = total - 1
	}
	end := start + visibleBlocks
	if end > total {
		end = total
	}
	if end < start {
		end = start
	}
	return start, end
}

func renderDashboardPaneTile(pane DashboardPane, width, height, previewLines int, selected bool, focused bool, iconCtx paneIconContext) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	layout := buildPaneTileLayout(width, height, previewLines, selected, focused)
	lines := paneTileBaseLines(pane, selected, iconCtx, layout.contentWidth)
	lines = append(lines, panePreviewLinesWithWidth(pane, layout.availablePreview, layout.contentWidth, false)...)
	return layout.style.Render(strings.Join(trimLines(lines, layout.contentHeight), "\n"))
}

func (m Model) renderDashboardPaneTileLive(pane DashboardPane, width, height, previewLines int, selected bool, focused bool, iconCtx paneIconContext) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	layout := buildPaneTileLayout(width, height, previewLines, selected, focused)
	lines := paneTileBaseLines(pane, selected, iconCtx, layout.contentWidth)
	lines = append(lines, paneLivePreviewLines(m, pane, layout, selected)...)
	return layout.style.Render(strings.Join(trimLines(lines, layout.contentHeight), "\n"))
}

type paneTileLayout struct {
	style            lipgloss.Style
	contentWidth     int
	contentHeight    int
	availablePreview int
}

func buildPaneTileLayout(width, height, previewLines int, selected bool, focused bool) paneTileLayout {
	if previewLines < 0 {
		previewLines = 0
	}
	borderColor := theme.Border
	if selected {
		borderColor = theme.BorderTarget
	}
	if focused {
		borderColor = theme.BorderFocus
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	frameW, frameH := style.GetFrameSize()
	borderW := style.GetHorizontalBorderSize()
	borderH := style.GetVerticalBorderSize()
	contentWidth := width - frameW
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeight := height - frameH
	if contentHeight < 1 {
		contentHeight = 1
	}
	blockWidth := width - borderW
	if blockWidth < 1 {
		blockWidth = 1
	}
	blockHeight := height - borderH
	if blockHeight < 1 {
		blockHeight = 1
	}
	style = style.Width(blockWidth).Height(blockHeight)
	availablePreview := previewLines
	if contentHeight-2 < availablePreview {
		availablePreview = contentHeight - 2
	}
	if availablePreview < 0 {
		availablePreview = 0
	}
	return paneTileLayout{
		style:            style,
		contentWidth:     contentWidth,
		contentHeight:    contentHeight,
		availablePreview: availablePreview,
	}
}

func paneTileBaseLines(pane DashboardPane, selected bool, iconCtx paneIconContext, contentWidth int) []string {
	header := paneTileHeader(pane, selected, iconCtx, contentWidth)
	detail := paneTileDetail(pane, selected, contentWidth)
	return []string{header, detail}
}

func paneTileHeader(pane DashboardPane, selected bool, iconCtx paneIconContext, contentWidth int) string {
	marker := " "
	if selected {
		marker = theme.SidebarCaret.Render(iconCtx.set.Caret.BySize(iconCtx.size))
	}
	label := fmt.Sprintf("%s / %s", pane.SessionName, paneLabel(pane.Pane))
	header := fmt.Sprintf("%s %s %s", marker, renderBadge(pane.Pane.Status), label)
	if selected {
		header = theme.SidebarSessionSelected.Render(header)
	}
	return truncateTileLine(header, contentWidth)
}

func paneTileDetail(pane DashboardPane, selected bool, contentWidth int) string {
	detail := strings.TrimSpace(pane.Pane.Command)
	if detail == "" {
		detail = strings.TrimSpace(pane.Pane.Title)
	}
	if detail == "" {
		detail = "-"
	}
	detail = "cmd: " + detail
	if selected {
		detail = theme.SidebarPaneSelected.Render(detail)
	} else {
		detail = theme.SidebarPane.Render(detail)
	}
	return truncateTileLine(detail, contentWidth)
}

func paneLivePreviewLines(m Model, pane DashboardPane, layout paneTileLayout, selected bool) []string {
	if layout.availablePreview <= 0 {
		return nil
	}
	if m.PaneView != nil && strings.TrimSpace(pane.Pane.ID) != "" {
		live := m.PaneView(pane.Pane.ID, layout.contentWidth, layout.availablePreview, selected && m.TerminalFocus)
		if live != "" {
			live = padLines(live, layout.contentWidth, layout.availablePreview)
			return strings.Split(live, "\n")
		}
	}
	return panePreviewLinesWithWidth(pane, layout.availablePreview, layout.contentWidth, true)
}

func panePreviewLinesWithWidth(pane DashboardPane, available, contentWidth int, useSummary bool) []string {
	if available <= 0 {
		return nil
	}
	preview := pane.Pane.Preview
	if useSummary && len(preview) == 0 {
		if summary := strings.TrimSpace(pane.Pane.SummaryLine); summary != "" {
			preview = []string{summary}
		}
	}
	preview = tailLines(preview, available)
	for len(preview) < available {
		preview = append(preview, "")
	}
	lines := make([]string, 0, available)
	for i := 0; i < available; i++ {
		lines = append(lines, truncateTileLine(preview[i], contentWidth))
	}
	return lines
}

func trimLines(lines []string, max int) []string {
	if max <= 0 {
		return nil
	}
	if len(lines) > max {
		return lines[:max]
	}
	return lines
}

func dashboardPaneIndex(panes []DashboardPane, selectionSession, selectionPane string) int {
	if len(panes) == 0 {
		return -1
	}
	if strings.TrimSpace(selectionSession) != "" {
		if selectionPane != "" {
			for i, pane := range panes {
				if pane.SessionName == selectionSession && pane.Pane.Index == selectionPane {
					return i
				}
			}
		}
		for i, pane := range panes {
			if pane.SessionName == selectionSession {
				return i
			}
		}
	}
	return -1
}
