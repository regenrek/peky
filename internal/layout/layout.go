package layout

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Grid represents a rectangular tmux pane arrangement.
type Grid struct {
	Rows    int
	Columns int
}

var (
	// Default is a balanced 2x2 workspace.
	Default = Grid{Rows: 2, Columns: 2}

	layoutRe = regexp.MustCompile(`^\s*(\d+)x(\d+)\s*$`)
)

// Parse converts strings like "2x3" into a Grid. Empty strings fall back to Default.
func Parse(spec string) (Grid, error) {
	if strings.TrimSpace(spec) == "" {
		return Default, nil
	}
	matches := layoutRe.FindStringSubmatch(spec)
	if len(matches) != 3 {
		return Grid{}, fmt.Errorf("invalid layout %q (expected <rows>x<columns>)", spec)
	}
	rows, err := strconv.Atoi(matches[1])
	if err != nil {
		return Grid{}, fmt.Errorf("invalid row count %q: %w", matches[1], err)
	}
	cols, err := strconv.Atoi(matches[2])
	if err != nil {
		return Grid{}, fmt.Errorf("invalid column count %q: %w", matches[2], err)
	}
	g := Grid{Rows: rows, Columns: cols}
	if err := g.Validate(); err != nil {
		return Grid{}, err
	}
	return g, nil
}

// Validate ensures the grid is reasonable for tmux panes.
func (g Grid) Validate() error {
	switch {
	case g.Rows <= 0:
		return fmt.Errorf("rows must be positive (got %d)", g.Rows)
	case g.Columns <= 0:
		return fmt.Errorf("columns must be positive (got %d)", g.Columns)
	case g.Rows*g.Columns > 12:
		return fmt.Errorf("layout %s creates %d panes; limit is 12 to keep panes readable", g, g.Rows*g.Columns)
	}
	return nil
}

// Panes returns the pane count for the grid.
func (g Grid) Panes() int {
	return g.Rows * g.Columns
}

func (g Grid) String() string {
	return fmt.Sprintf("%dx%d", g.Rows, g.Columns)
}

// CommonPresets returns human-friendly presets for the interactive picker.
func CommonPresets() []Grid {
	return []Grid{
		{Rows: 1, Columns: 2},
		{Rows: 1, Columns: 3},
		{Rows: 2, Columns: 2},
		{Rows: 2, Columns: 3},
		{Rows: 3, Columns: 2},
		{Rows: 3, Columns: 3},
	}
}
