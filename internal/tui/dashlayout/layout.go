package dashlayout

const (
	DefaultColumnGap = 2
	MinColumnWidth   = 24
	MinBlockHeight   = 3
)

// ClampColumns returns a window [start,end) into total columns and the adjusted selected index.
func ClampColumns(total, selectedIndex, width, gap int) (start, end, adjustedSelected int) {
	if total <= 0 {
		return 0, 0, 0
	}
	if gap < 0 {
		gap = 0
	}
	maxCols := (width + gap) / (MinColumnWidth + gap)
	if maxCols < 1 {
		maxCols = 1
	}
	if total <= maxCols {
		return 0, total, selectedIndex
	}
	start = selectedIndex - maxCols/2
	if start < 0 {
		start = 0
	}
	if start+maxCols > total {
		start = total - maxCols
	}
	end = start + maxCols
	adjustedSelected = selectedIndex - start
	return start, end, adjustedSelected
}

// ColumnWidth returns the width for each column when splitting total width.
func ColumnWidth(width, columns, gap int) int {
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

// PaneBlockHeight returns the unbounded block height for a pane preview.
func PaneBlockHeight(previewLines int) int {
	if previewLines < 0 {
		previewLines = 0
	}
	return previewLines + 4
}

// BlockHeight clamps the block height to the available body height.
func BlockHeight(previewLines, bodyHeight int) int {
	if bodyHeight <= 0 {
		return 0
	}
	blockHeight := PaneBlockHeight(previewLines)
	if blockHeight > bodyHeight {
		blockHeight = bodyHeight
	}
	if blockHeight < MinBlockHeight {
		blockHeight = bodyHeight
	}
	return blockHeight
}

// VisibleBlocks returns the number of visible blocks given a body and block height.
func VisibleBlocks(bodyHeight, blockHeight int) int {
	if bodyHeight <= 0 || blockHeight <= 0 {
		return 0
	}
	visible := bodyHeight / blockHeight
	if visible < 1 {
		visible = 1
	}
	return visible
}

// PaneRange returns the visible range of panes.
func PaneRange(selected bool, selectedIndex, visibleBlocks, total int) (int, int) {
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
