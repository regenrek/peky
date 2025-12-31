package limits

import "fmt"

const (
	PaneMaxCols = 500
	PaneMaxRows = 200
)

type DimensionError struct {
	Cols, Rows       int
	MaxCols, MaxRows int
}

func (e *DimensionError) Error() string {
	return fmt.Sprintf("dimensions %dx%d exceed max %dx%d", e.Cols, e.Rows, e.MaxCols, e.MaxRows)
}

func Normalize(cols, rows int) (int, int) {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

func Clamp(cols, rows int) (int, int) {
	cols, rows = Normalize(cols, rows)
	if cols > PaneMaxCols {
		cols = PaneMaxCols
	}
	if rows > PaneMaxRows {
		rows = PaneMaxRows
	}
	return cols, rows
}

func ValidateMax(cols, rows int) error {
	cols, rows = Normalize(cols, rows)
	if cols > PaneMaxCols || rows > PaneMaxRows {
		return &DimensionError{Cols: cols, Rows: rows, MaxCols: PaneMaxCols, MaxRows: PaneMaxRows}
	}
	return nil
}
