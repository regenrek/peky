package views

import (
	"fmt"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/ansi"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

type layoutPreviewContext struct {
	targetPane    string
	paneCursor    bool
	guides        []ResizeGuide
	paneView      func(id string, width, height int, showCursor bool) string
	paneTopbar    bool
	topbarSpinner string
}

func renderPaneLayout(panes []Pane, width, height int, ctx layoutPreviewContext) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(panes) == 0 {
		return padLines("No panes", width, height)
	}
	geom, paneByID, ok := buildPaneLayoutGeometry(panes, width, height)
	if !ok {
		return padLines("No panes", width, height)
	}

	buf := newPaneLayoutBuffer(width, height)
	renderPaneLayoutContent(buf, geom, paneByID, ctx)
	dividerMap := drawPaneLayoutDividers(buf, geom, width, height)
	highlightTargetPaneBorder(buf, geom, paneByID, dividerMap, ctx)
	if len(ctx.guides) > 0 && len(dividerMap) > 0 {
		applyResizeGuideStyles(buf, dividerMap, ctx.guides)
	}
	return renderBufferLines(buf)
}

func buildPaneLayoutGeometry(panes []Pane, width, height int) (layoutgeom.Geometry, map[string]Pane, bool) {
	if width <= 0 || height <= 0 || len(panes) == 0 {
		return layoutgeom.Geometry{}, nil, false
	}
	preview := mouse.Rect{X: 0, Y: 0, W: width, H: height}
	rects := make(map[string]layout.Rect, len(panes))
	paneByID := make(map[string]Pane, len(panes))
	for _, pane := range panes {
		if pane.ID == "" || pane.Width <= 0 || pane.Height <= 0 {
			continue
		}
		rects[pane.ID] = layout.Rect{X: pane.Left, Y: pane.Top, W: pane.Width, H: pane.Height}
		paneByID[pane.ID] = pane
	}
	geom, ok := layoutgeom.Build(preview, rects)
	if !ok {
		return layoutgeom.Geometry{}, nil, false
	}
	return geom, paneByID, true
}

func newPaneLayoutBuffer(width, height int) *cellbuf.Buffer {
	buf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(buf, padLines("", width, height))
	return buf
}

func renderPaneLayoutContent(buf *cellbuf.Buffer, geom layoutgeom.Geometry, paneByID map[string]Pane, ctx layoutPreviewContext) {
	if buf == nil || len(paneByID) == 0 {
		return
	}
	for _, paneGeom := range geom.Panes {
		pane, ok := paneByID[paneGeom.ID]
		if !ok {
			continue
		}
		outer := paneGeom.Screen
		if outer.Empty() {
			continue
		}
		rect := layoutgeom.ContentRect(geom, outer)
		if rect.Empty() {
			continue
		}
		topbarEnabled := ctx.paneTopbar && rect.H >= 2
		if topbarEnabled {
			topbar := renderPaneTopbar(pane, rect.W, ctx.topbarSpinner)
			topbar = theme.PaneTopbar.Render(fitLine(topbar, rect.W))
			cellbuf.SetContentRect(buf, topbar, cellbuf.Rect(rect.X, rect.Y, rect.W, 1))
		}
		contentRect := rect
		if topbarEnabled {
			contentRect = mouse.Rect{X: rect.X, Y: rect.Y + 1, W: rect.W, H: rect.H - 1}
		}
		content := strings.TrimSpace(layoutPaneContent(pane, contentRect.W, contentRect.H, ctx))
		content = padLines(content, contentRect.W, contentRect.H)
		cellbuf.SetContentRect(buf, content, cellbuf.Rect(contentRect.X, contentRect.Y, contentRect.W, contentRect.H))
		if bg, ok := paneBackgroundAnsiTint(pane.Background); ok {
			applyPaneBackground(buf, contentRect, bg)
		}
	}
}

func drawPaneLayoutDividers(buf *cellbuf.Buffer, geom layoutgeom.Geometry, width, height int) map[[2]int]rune {
	dividerCells := layoutgeom.DividerCells(geom.Dividers)
	dividerMap := make(map[[2]int]rune, len(dividerCells))
	if buf == nil || width <= 0 || height <= 0 {
		return dividerMap
	}
	for _, cell := range dividerCells {
		if cell.X < 0 || cell.Y < 0 || cell.X >= width || cell.Y >= height {
			continue
		}
		buf.SetCell(cell.X, cell.Y, cellbuf.NewCell(cell.Rune))
		dividerMap[[2]int{cell.X, cell.Y}] = cell.Rune
	}
	return dividerMap
}

func highlightTargetPaneBorder(buf *cellbuf.Buffer, geom layoutgeom.Geometry, paneByID map[string]Pane, dividerMap map[[2]int]rune, ctx layoutPreviewContext) {
	if buf == nil || ctx.targetPane == "" || len(paneByID) == 0 || len(dividerMap) == 0 {
		return
	}
	highlight := paneHighlightStyle(false)
	for _, paneGeom := range geom.Panes {
		if paneGeom.ID == "" {
			continue
		}
		pane, ok := paneByID[paneGeom.ID]
		if !ok || pane.Index != ctx.targetPane {
			continue
		}
		applyPaneBorderHighlight(buf, geom, paneGeom, dividerMap, highlight)
		return
	}
}

func layoutPaneContent(pane Pane, width, height int, ctx layoutPreviewContext) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if ctx.paneView != nil && strings.TrimSpace(pane.ID) != "" {
		showCursor := pane.Index == ctx.targetPane && ctx.paneCursor
		if content := ctx.paneView(pane.ID, width, height, showCursor); strings.TrimSpace(content) != "" {
			return content
		}
	}
	lines := layoutFallbackLines(pane, ctx.targetPane)
	return strings.Join(lines, "\n")
}

func layoutFallbackLines(pane Pane, targetPane string) []string {
	title := pane.Title
	if pane.Index == targetPane {
		title = "TARGET " + title
	}
	command := pane.Command
	status := ansi.LastNonEmpty(pane.Preview)
	if status == "" {
		status = "idle"
	}
	return []string{title, command, status}
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

func paneHighlightStyle(focused bool) cellbuf.Style {
	color := xansi.XParseColor(string(theme.BorderTarget))
	if focused {
		color = xansi.XParseColor(string(theme.BorderFocus))
	}
	return cellbuf.Style{Fg: color}
}

func applyPaneBorderHighlight(buf *cellbuf.Buffer, geom layoutgeom.Geometry, pane layoutgeom.Pane, dividerMap map[[2]int]rune, style cellbuf.Style) {
	if buf == nil || len(dividerMap) == 0 || pane.ID == "" {
		return
	}

	edges := []sessiond.ResizeEdge{
		sessiond.ResizeEdgeLeft,
		sessiond.ResizeEdgeRight,
		sessiond.ResizeEdgeUp,
		sessiond.ResizeEdgeDown,
	}
	for _, edge := range edges {
		rect, ok := layoutgeom.EdgeLineRect(geom, layoutgeom.EdgeRef{PaneID: pane.ID, Edge: edge})
		if !ok || rect.Empty() {
			continue
		}
		x0 := rect.X
		y0 := rect.Y
		x1 := rect.X + rect.W - 1
		y1 := rect.Y + rect.H - 1
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				key := [2]int{x, y}
				r, ok := dividerMap[key]
				if !ok {
					continue
				}
				cell := cellbuf.NewCell(r)
				cell.Style = style
				buf.SetCell(x, y, cell)
			}
		}
	}
}

func resizeGuideStyle(active bool) cellbuf.Style {
	color := xansi.XParseColor(string(theme.Accent))
	if active {
		color = xansi.XParseColor(string(theme.AccentFocus))
	}
	return cellbuf.Style{Fg: color}
}

func applyPaneBackground(buf *cellbuf.Buffer, rect mouse.Rect, bg xansi.Color) {
	if buf == nil || rect.Empty() || bg == nil {
		return
	}
	for y := rect.Y; y < rect.Y+rect.H; y++ {
		for x := rect.X; x < rect.X+rect.W; x++ {
			cell := buf.Cell(x, y)
			if cell == nil {
				blank := cellbuf.BlankCell
				blank.Style.Bg = bg
				buf.SetCell(x, y, &blank)
				continue
			}
			if cell.Style.Bg != nil {
				continue
			}
			cell.Style.Bg = bg
			buf.SetCell(x, y, cell)
		}
	}
}

func applyResizeGuideStyles(buf *cellbuf.Buffer, dividerMap map[[2]int]rune, guides []ResizeGuide) {
	if buf == nil || len(dividerMap) == 0 || len(guides) == 0 {
		return
	}
	for _, guide := range guides {
		if guide.W <= 0 || guide.H <= 0 {
			continue
		}
		style := resizeGuideStyle(guide.Active)
		x0 := guide.X
		y0 := guide.Y
		x1 := guide.X + guide.W - 1
		y1 := guide.Y + guide.H - 1
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				key := [2]int{x, y}
				r, ok := dividerMap[key]
				if !ok {
					continue
				}
				cell := cellbuf.NewCell(r)
				cell.Style = style
				buf.SetCell(x, y, cell)
			}
		}
		applyResizeGuideHandle(buf, dividerMap, guide, style)
	}
}

func applyResizeGuideHandle(buf *cellbuf.Buffer, dividerMap map[[2]int]rune, guide ResizeGuide, style cellbuf.Style) {
	if buf == nil || len(dividerMap) == 0 {
		return
	}
	if guide.W <= 0 || guide.H <= 0 {
		return
	}

	switch {
	case guide.W == 1 && guide.H >= 3:
		x := guide.X
		y := pickNonJunctionY(dividerMap, x, guide.Y, guide.H)
		if y < 0 {
			return
		}
		setGuideHandleCell(buf, dividerMap, x, y, '↔', style)
	case guide.H == 1 && guide.W >= 3:
		y := guide.Y
		x := pickNonJunctionX(dividerMap, y, guide.X, guide.W)
		if x < 0 {
			return
		}
		setGuideHandleCell(buf, dividerMap, x, y, '↕', style)
	}
}

func pickNonJunctionY(dividerMap map[[2]int]rune, x, y0, h int) int {
	if h <= 0 {
		return -1
	}
	mid := y0 + h/2
	for delta := 0; delta <= h/2; delta++ {
		for _, y := range []int{mid + delta, mid - delta} {
			if y < y0 || y >= y0+h {
				continue
			}
			r, ok := dividerMap[[2]int{x, y}]
			if !ok || isJunctionRune(r) {
				continue
			}
			return y
		}
	}
	return -1
}

func pickNonJunctionX(dividerMap map[[2]int]rune, y, x0, w int) int {
	if w <= 0 {
		return -1
	}
	mid := x0 + w/2
	for delta := 0; delta <= w/2; delta++ {
		for _, x := range []int{mid + delta, mid - delta} {
			if x < x0 || x >= x0+w {
				continue
			}
			r, ok := dividerMap[[2]int{x, y}]
			if !ok || isJunctionRune(r) {
				continue
			}
			return x
		}
	}
	return -1
}

func isJunctionRune(r rune) bool {
	switch r {
	case '┼', '┬', '┴', '├', '┤', '┌', '┐', '└', '┘':
		return true
	default:
		return false
	}
}

func setGuideHandleCell(buf *cellbuf.Buffer, dividerMap map[[2]int]rune, x, y int, handle rune, style cellbuf.Style) {
	if buf == nil {
		return
	}
	if _, ok := dividerMap[[2]int{x, y}]; !ok {
		return
	}
	cell := cellbuf.NewCell(handle)
	cell.Style = style
	buf.SetCell(x, y, cell)
}
