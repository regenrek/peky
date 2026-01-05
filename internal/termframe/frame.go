package termframe

import (
	"image/color"

	xansi "github.com/charmbracelet/x/ansi"
)

// ColorKind describes how a color is encoded.
type ColorKind uint8

const (
	ColorNone ColorKind = iota
	ColorBasic
	ColorIndexed
	ColorRGB
)

// Color stores a terminal color in a compact, serializable form.
// Value encodes Basic/Indexed as their index, and RGB as 0xRRGGBB.
type Color struct {
	Kind  ColorKind
	Value uint32
}

func (c Color) IsZero() bool {
	return c.Kind == ColorNone
}

// ColorFromColor converts a color.Color into a termframe Color.
func ColorFromColor(c color.Color) Color {
	if c == nil {
		return Color{}
	}
	switch v := c.(type) {
	case xansi.BasicColor:
		return Color{Kind: ColorBasic, Value: uint32(v)}
	case xansi.IndexedColor:
		return Color{Kind: ColorIndexed, Value: uint32(v)}
	case xansi.RGBColor:
		return Color{
			Kind:  ColorRGB,
			Value: uint32(v.R)<<16 | uint32(v.G)<<8 | uint32(v.B),
		}
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return Color{}
	}
	return Color{
		Kind:  ColorRGB,
		Value: uint32(r>>8)<<16 | uint32(g>>8)<<8 | uint32(b>>8),
	}
}

// Attrs are text attributes applied to a cell.
type Attrs uint16

const (
	AttrBold Attrs = 1 << iota
	AttrFaint
	AttrItalic
	AttrBlink
	AttrRapidBlink
	AttrReverse
	AttrConceal
	AttrStrikethrough
)

// UnderlineStyle represents the underline style.
type UnderlineStyle uint8

const (
	UnderlineNone UnderlineStyle = iota
	UnderlineSingle
	UnderlineDouble
	UnderlineCurly
	UnderlineDotted
	UnderlineDashed
)

// Style represents the style of a cell.
type Style struct {
	Fg             Color
	Bg             Color
	UnderlineColor Color
	UnderlineStyle UnderlineStyle
	Attrs          Attrs
}

func (s Style) IsZero() bool {
	return s.Attrs == 0 &&
		s.UnderlineStyle == UnderlineNone &&
		s.Fg.IsZero() &&
		s.Bg.IsZero() &&
		s.UnderlineColor.IsZero()
}

// Link represents a hyperlink.
type Link struct {
	URL    string
	Params string
}

func (l Link) IsZero() bool {
	return l.URL == "" && l.Params == ""
}

// Cell represents a terminal cell.
type Cell struct {
	Content string
	Width   int
	Style   Style
	Link    Link
}

func (c Cell) IsZero() bool {
	return c.Content == "" && c.Width == 0 && c.Style.IsZero() && c.Link.IsZero()
}

// Cursor represents the terminal cursor state.
type Cursor struct {
	X       int
	Y       int
	Visible bool
}

// Frame is a snapshot of terminal cells.
type Frame struct {
	Cols   int
	Rows   int
	Cells  []Cell
	Cursor Cursor
}

func (f Frame) Empty() bool {
	return f.Cols <= 0 || f.Rows <= 0 || len(f.Cells) == 0
}

func (f Frame) CellAt(x, y int) *Cell {
	if x < 0 || y < 0 || x >= f.Cols || y >= f.Rows {
		return nil
	}
	idx := y*f.Cols + x
	if idx < 0 || idx >= len(f.Cells) {
		return nil
	}
	return &f.Cells[idx]
}
