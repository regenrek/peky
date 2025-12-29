package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/ansi"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

type paneTileRenderer func(pane Pane, width, height int, compact bool, target bool, borders tileBorders) string

func renderPanePreview(panes []Pane, width, height int, mode string, compact bool, targetPane string, terminalFocus bool) string {
	return renderPanePreviewWithRenderer(panes, width, height, panePreviewContext{
		mode:          mode,
		compact:       compact,
		targetPane:    targetPane,
		terminalFocus: terminalFocus,
	})
}

type panePreviewContext struct {
	mode          string
	compact       bool
	targetPane    string
	terminalFocus bool
	renderer      paneTileRenderer
}

func renderPanePreviewWithRenderer(panes []Pane, width, height int, ctx panePreviewContext) string {
	if ctx.mode == "layout" {
		return renderPaneLayout(panes, width, height, ctx.targetPane)
	}
	return renderPaneTilesWithRenderer(panes, width, height, ctx.compact, ctx.targetPane, ctx.terminalFocus, ctx.renderer)
}

func renderPaneLayout(panes []Pane, width, height int, targetPane string) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(panes) == 0 {
		return padLines("No panes", width, height)
	}
	c := newCanvas(width, height)

	maxW, maxH := paneBounds(panes)
	if maxW == 0 || maxH == 0 {
		return padLines("No panes", width, height)
	}

	for _, pane := range panes {
		x1, y1, w, h := scalePane(pane, maxW, maxH, width, height)
		if w < 2 || h < 2 {
			continue
		}
		c.drawBox(x1, y1, w, h)
		title := pane.Title
		if pane.Index == targetPane {
			title = "TARGET " + title
		}
		if pane.Active {
			title = "▶ " + title
		}
		c.write(x1+1, y1+1, title, w-2)
		c.write(x1+1, y1+2, pane.Command, w-2)
		status := ansi.LastNonEmpty(pane.Preview)
		if status == "" {
			status = "idle"
		}
		c.write(x1+1, y1+3, status, w-2)
	}

	return c.String()
}

const (
	borderLevelDefault = iota
	borderLevelActive
	borderLevelTarget
	borderLevelFocus
)

func borderLevelForPane(pane Pane, targetPane string, terminalFocus bool) int {
	if pane.Index == targetPane {
		if terminalFocus {
			return borderLevelFocus
		}
		return borderLevelTarget
	}
	if pane.Active {
		return borderLevelActive
	}
	return borderLevelDefault
}

func borderColorFor(level int) lipgloss.TerminalColor {
	switch level {
	case borderLevelTarget:
		return theme.BorderTarget
	case borderLevelFocus:
		return theme.BorderFocus
	case borderLevelActive:
		return theme.BorderFocused
	default:
		return theme.Border
	}
}

func maxBorderLevel(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func renderPaneTilesWithRenderer(panes []Pane, width, height int, compact bool, targetPane string, terminalFocus bool, renderer paneTileRenderer) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(panes) == 0 {
		return padLines("No panes", width, height)
	}
	if renderer == nil {
		renderer = renderPaneTile
	}

	layout := computePaneGridLayout(len(panes), width, height)
	paneLevels := paneBorderLevels(panes, targetPane, terminalFocus)
	ctx := paneTileContext{
		panes:      panes,
		layout:     layout,
		paneLevels: paneLevels,
		renderer:   renderer,
		compact:    compact,
		targetPane: targetPane,
	}

	renderedRows := make([]string, 0, layout.rows)
	for r := 0; r < layout.rows; r++ {
		rowHeight := paneRowHeight(layout, r)
		tiles := ctx.renderRow(r, rowHeight)
		row := lipgloss.JoinHorizontal(lipgloss.Top, tiles...)
		renderedRows = append(renderedRows, row)
	}
	return padLines(strings.Join(renderedRows, "\n"), width, height)
}

type paneGridLayout struct {
	cols        int
	rows        int
	tileWidth   int
	baseHeight  int
	extraHeight int
}

func computePaneGridLayout(paneCount, width, height int) paneGridLayout {
	cols := 3
	if width < 70 {
		cols = 2
	}
	if width < 42 {
		cols = 1
	}
	if paneCount < cols {
		cols = paneCount
	}
	if cols <= 0 {
		cols = 1
	}

	rows := (paneCount + cols - 1) / cols
	availableHeight := height
	if availableHeight < rows {
		availableHeight = rows
	}
	baseHeight := availableHeight / rows
	extraHeight := availableHeight % rows
	if baseHeight < 4 {
		baseHeight = 4
		extraHeight = 0
	}
	tileWidth := width / cols
	if tileWidth < 14 {
		tileWidth = 14
	}

	return paneGridLayout{
		cols:        cols,
		rows:        rows,
		tileWidth:   tileWidth,
		baseHeight:  baseHeight,
		extraHeight: extraHeight,
	}
}

func paneBorderLevels(panes []Pane, targetPane string, terminalFocus bool) []int {
	levels := make([]int, len(panes))
	for i, pane := range panes {
		levels[i] = borderLevelForPane(pane, targetPane, terminalFocus)
	}
	return levels
}

func paneRowHeight(layout paneGridLayout, row int) int {
	if row == layout.rows-1 {
		return layout.baseHeight + layout.extraHeight
	}
	return layout.baseHeight
}

type paneTileContext struct {
	panes      []Pane
	layout     paneGridLayout
	paneLevels []int
	renderer   paneTileRenderer
	compact    bool
	targetPane string
}

func (ctx paneTileContext) renderRow(row, rowHeight int) []string {
	tiles := make([]string, 0, ctx.layout.cols)
	for c := 0; c < ctx.layout.cols; c++ {
		idx := row*ctx.layout.cols + c
		if idx >= len(ctx.panes) {
			tiles = append(tiles, padLines("", ctx.layout.tileWidth, rowHeight))
			continue
		}
		borders := paneTileBorders(ctx.layout, ctx.paneLevels, idx, row, c)
		tiles = append(tiles, ctx.renderer(ctx.panes[idx], ctx.layout.tileWidth, rowHeight, ctx.compact, ctx.panes[idx].Index == ctx.targetPane, borders))
	}
	return tiles
}

func paneTileBorders(layout paneGridLayout, paneLevels []int, idx, row, col int) tileBorders {
	level := paneLevels[idx]
	rightLevel := borderLevelDefault
	if col < layout.cols-1 {
		neighbor := idx + 1
		if neighbor < len(paneLevels) {
			rightLevel = paneLevels[neighbor]
		}
	}
	bottomLevel := borderLevelDefault
	if row < layout.rows-1 {
		neighbor := idx + layout.cols
		if neighbor < len(paneLevels) {
			bottomLevel = paneLevels[neighbor]
		}
	}
	colors := tileBorderColors{
		top:    borderColorFor(level),
		left:   borderColorFor(level),
		right:  borderColorFor(maxBorderLevel(level, rightLevel)),
		bottom: borderColorFor(maxBorderLevel(level, bottomLevel)),
	}
	return tileBorders{
		top:    row == 0,
		left:   col == 0,
		right:  true,
		bottom: true,
		colors: colors,
	}
}

type tileBorderColors struct {
	top    lipgloss.TerminalColor
	right  lipgloss.TerminalColor
	bottom lipgloss.TerminalColor
	left   lipgloss.TerminalColor
}

type tileBorders struct {
	top    bool
	right  bool
	bottom bool
	left   bool
	colors tileBorderColors
}

func renderPaneTile(pane Pane, width, height int, compact bool, target bool, borders tileBorders) string {
	title := pane.Title
	if target {
		title = "TARGET " + title
	}
	if pane.Active {
		title = "▶ " + title
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(borders.top).
		BorderRight(borders.right).
		BorderBottom(borders.bottom).
		BorderLeft(borders.left).
		BorderForeground(theme.Border).
		BorderTopForeground(borders.colors.top).
		BorderRightForeground(borders.colors.right).
		BorderBottomForeground(borders.colors.bottom).
		BorderLeftForeground(borders.colors.left).
		Padding(0, 1)

	frameW, frameH := style.GetFrameSize()
	borderW := style.GetHorizontalBorderSize()
	borderH := style.GetVerticalBorderSize()
	contentWidth := width - frameW
	if contentWidth < 1 {
		contentWidth = 1
	}
	innerHeight := height - frameH
	if innerHeight < 1 {
		innerHeight = 1
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

	header := fmt.Sprintf("%s %s", renderBadge(pane.Status), title)
	lines := []string{truncateTileLine(header, contentWidth)}
	if strings.TrimSpace(pane.Command) != "" {
		lines = append(lines, truncateTileLine(pane.Command, contentWidth))
	}

	previewSource := pane.Preview
	if compact {
		previewSource = compactPreviewLines(previewSource)
	}
	previewSource = trimTrailingBlankLines(previewSource)

	maxPreview := innerHeight - len(lines)
	if maxPreview < 0 {
		maxPreview = 0
	}
	previewLines := tailLines(previewSource, maxPreview)
	lines = append(lines, truncateTileLines(previewLines, contentWidth)...)

	content := strings.Join(lines, "\n")
	return style.Render(content)
}

func (m Model) renderPaneTileLive(pane Pane, width, height int, compact bool, target bool, borders tileBorders) string {
	title := pane.Title
	if target {
		title = "TARGET " + title
	}
	if pane.Active {
		title = "▶ " + title
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(borders.top).
		BorderRight(borders.right).
		BorderBottom(borders.bottom).
		BorderLeft(borders.left).
		BorderForeground(theme.Border).
		BorderTopForeground(borders.colors.top).
		BorderRightForeground(borders.colors.right).
		BorderBottomForeground(borders.colors.bottom).
		BorderLeftForeground(borders.colors.left).
		Padding(0, 1)

	frameW, frameH := style.GetFrameSize()
	borderW := style.GetHorizontalBorderSize()
	borderH := style.GetVerticalBorderSize()
	contentWidth := width - frameW
	if contentWidth < 1 {
		contentWidth = 1
	}
	innerHeight := height - frameH
	if innerHeight < 1 {
		innerHeight = 1
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

	header := fmt.Sprintf("%s %s", renderBadge(pane.Status), title)
	lines := []string{truncateTileLine(header, contentWidth)}
	if strings.TrimSpace(pane.Command) != "" {
		lines = append(lines, truncateTileLine(pane.Command, contentWidth))
	}

	maxPreview := innerHeight - len(lines)
	if maxPreview < 0 {
		maxPreview = 0
	}
	if maxPreview > 0 {
		live := ""
		if m.PaneView != nil && strings.TrimSpace(pane.ID) != "" {
			live = m.PaneView(pane.ID, contentWidth, maxPreview, target && m.TerminalFocus)
		}
		if live != "" {
			live = padLines(live, contentWidth, maxPreview)
			lines = append(lines, strings.Split(live, "\n")...)
		} else {
			previewSource := pane.Preview
			if len(previewSource) == 0 {
				if summary := strings.TrimSpace(pane.SummaryLine); summary != "" {
					previewSource = []string{summary}
				}
			}
			if compact {
				previewSource = compactPreviewLines(previewSource)
			}
			previewSource = trimTrailingBlankLines(previewSource)
			previewLines := tailLines(previewSource, maxPreview)
			lines = append(lines, truncateTileLines(previewLines, contentWidth)...)
		}
	}

	content := padLines(strings.Join(lines, "\n"), contentWidth, innerHeight)
	return style.Render(content)
}

func paneBounds(panes []Pane) (int, int) {
	maxW := 0
	maxH := 0
	for _, p := range panes {
		if p.Left+p.Width > maxW {
			maxW = p.Left + p.Width
		}
		if p.Top+p.Height > maxH {
			maxH = p.Top + p.Height
		}
	}
	return maxW, maxH
}

func scalePane(p Pane, totalW, totalH, width, height int) (int, int, int, int) {
	x1 := int(float64(p.Left) / float64(totalW) * float64(width))
	y1 := int(float64(p.Top) / float64(totalH) * float64(height))
	x2 := int(float64(p.Left+p.Width) / float64(totalW) * float64(width))
	y2 := int(float64(p.Top+p.Height) / float64(totalH) * float64(height))
	w := x2 - x1
	h := y2 - y1
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}
	if x1+w > width {
		w = width - x1
	}
	if y1+h > height {
		h = height - y1
	}
	return x1, y1, w, h
}

func paneLabel(pane Pane) string {
	label := strings.TrimSpace(pane.Title)
	if label == "" {
		label = strings.TrimSpace(pane.Command)
	}
	if label == "" {
		return fmt.Sprintf("pane %s", pane.Index)
	}
	if strings.TrimSpace(pane.Index) == "" {
		return label
	}
	return fmt.Sprintf("%s %s", pane.Index, label)
}
