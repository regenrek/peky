package termframe

import "github.com/mattn/go-runewidth"

// FrameFromLines builds a frame from plain text lines.
// Lines are rendered top-to-bottom and truncated to the provided dimensions.
func FrameFromLines(cols, rows int, lines []string) Frame {
	if cols <= 0 || rows <= 0 {
		return Frame{}
	}
	frame := Frame{
		Cols:  cols,
		Rows:  rows,
		Cells: make([]Cell, cols*rows),
	}
	for i := range frame.Cells {
		frame.Cells[i] = Cell{Content: " ", Width: 1}
	}
	maxLines := rows
	if len(lines) < maxLines {
		maxLines = len(lines)
	}
	for y := 0; y < maxLines; y++ {
		line := lines[y]
		x := 0
		for _, r := range line {
			if x >= cols {
				break
			}
			width := runewidth.RuneWidth(r)
			if width <= 0 {
				width = 1
			}
			if x+width > cols {
				break
			}
			idx := y*cols + x
			frame.Cells[idx] = Cell{Content: string(r), Width: width}
			for i := 1; i < width; i++ {
				if x+i >= cols {
					break
				}
				frame.Cells[y*cols+x+i] = Cell{Width: 0}
			}
			x += width
		}
	}
	return frame
}
