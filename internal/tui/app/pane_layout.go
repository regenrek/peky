package app

func dashboardPaneBlockHeight(previewLines int) int {
	if previewLines < 0 {
		previewLines = 0
	}
	return previewLines + 4
}

func paneBounds(panes []PaneItem) (int, int) {
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

func scalePane(p PaneItem, totalW, totalH, width, height int) (int, int, int, int) {
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
