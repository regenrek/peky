package panelayout

type TileBorders struct {
	Top    bool
	Left   bool
	Right  bool
	Bottom bool
}

type TileMetrics struct {
	ContentX     int
	ContentY     int
	ContentWidth int
	InnerHeight  int
}

const (
	tilePadLeft   = 1
	tilePadRight  = 1
	tilePadTop    = 0
	tilePadBottom = 0
)

func BorderSizes(borders TileBorders) (int, int) {
	return boolToInt(borders.Left) + boolToInt(borders.Right),
		boolToInt(borders.Top) + boolToInt(borders.Bottom)
}

func TileMetricsFor(width, height int, borders TileBorders) TileMetrics {
	left := boolToInt(borders.Left)
	right := boolToInt(borders.Right)
	top := boolToInt(borders.Top)
	bottom := boolToInt(borders.Bottom)

	contentX := left + tilePadLeft
	contentY := top + tilePadTop
	contentWidth := width - left - right - tilePadLeft - tilePadRight
	innerHeight := height - top - bottom - tilePadTop - tilePadBottom

	if contentWidth < 0 {
		contentWidth = 0
	}
	if innerHeight < 0 {
		innerHeight = 0
	}

	return TileMetrics{
		ContentX:     contentX,
		ContentY:     contentY,
		ContentWidth: contentWidth,
		InnerHeight:  innerHeight,
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
