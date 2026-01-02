package panelayout

type Grid struct {
	Cols        int
	Rows        int
	TileWidth   int
	BaseHeight  int
	ExtraHeight int
}

func Compute(paneCount, width, height int) Grid {
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
	tileWidth := 0
	if cols > 0 {
		tileWidth = width / cols
	}
	if tileWidth < 14 {
		tileWidth = 14
	}

	return Grid{
		Cols:        cols,
		Rows:        rows,
		TileWidth:   tileWidth,
		BaseHeight:  baseHeight,
		ExtraHeight: extraHeight,
	}
}

func (g Grid) RowHeight(row int) int {
	if row == g.Rows-1 {
		return g.BaseHeight + g.ExtraHeight
	}
	return g.BaseHeight
}

func (g Grid) RowY(originY, row int) int {
	y := originY
	for i := 0; i < row; i++ {
		y += g.RowHeight(i)
	}
	return y
}
