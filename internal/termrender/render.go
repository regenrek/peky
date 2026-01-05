package termrender

import (
	"strings"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/termframe"
)

// Options configures frame rendering.
type Options struct {
	Profile    colorprofile.Profile
	ShowCursor bool
}

// Render converts a frame into an ANSI string using the requested profile.
func Render(frame termframe.Frame, opts Options) string {
	if frame.Cols <= 0 || frame.Rows <= 0 || len(frame.Cells) == 0 {
		return ""
	}
	profile := normalizeProfile(opts.Profile)

	cursorActive := opts.ShowCursor && frame.Cursor.Visible

	var b strings.Builder
	b.Grow(frame.Cols * frame.Rows)
	for y := 0; y < frame.Rows; y++ {
		renderLine(&b, frame, y, profile, cursorActive)
		if y < frame.Rows-1 {
			_ = b.WriteByte('\n')
		}
	}
	return b.String()
}

func normalizeProfile(profile colorprofile.Profile) colorprofile.Profile {
	switch profile {
	case colorprofile.TrueColor, colorprofile.ANSI256, colorprofile.ANSI, colorprofile.ASCII, colorprofile.NoTTY:
		return profile
	default:
		return colorprofile.TrueColor
	}
}

func renderLine(b *strings.Builder, frame termframe.Frame, y int, profile colorprofile.Profile, cursorActive bool) {
	var pen uv.Style
	var link termframe.Link
	pendingSpaces := 0

	for x := 0; x < frame.Cols; {
		cell := frame.CellAt(x, y)
		if cell == nil {
			x++
			continue
		}
		if cell.Width == 0 {
			x++
			continue
		}

		if cellIsEmpty(*cell) {
			if pendingSpaces == 0 {
				renderResetForSpace(b, &pen, &link)
			}
			pendingSpaces++
			x++
			continue
		}

		if pendingSpaces > 0 {
			renderSpaces(b, pendingSpaces)
			pendingSpaces = 0
		}

		style := styleFromFrame(cell.Style, profile)
		if cursorActive && x == frame.Cursor.X && y == frame.Cursor.Y {
			style.Attrs |= uv.AttrReverse | uv.AttrBold
		}
		renderApplyStyle(b, &pen, style)
		renderApplyLink(b, &link, cell.Link)
		b.WriteString(cellContent(*cell))
		if cell.Width > 1 {
			x += cell.Width
		} else {
			x++
		}
	}
	if pendingSpaces > 0 {
		renderSpaces(b, pendingSpaces)
	}
	renderFinalizeLine(b, &pen, &link)
}

func cellIsEmpty(cell termframe.Cell) bool {
	if !cell.Style.IsZero() || !cell.Link.IsZero() {
		return false
	}
	return cell.Content == "" || cell.Content == " "
}

func cellContent(cell termframe.Cell) string {
	if cell.Content == "" {
		return " "
	}
	return cell.Content
}

func styleFromFrame(s termframe.Style, profile colorprofile.Profile) uv.Style {
	return uv.Style{
		Fg:             profile.Convert(colorFromFrame(s.Fg)),
		Bg:             profile.Convert(colorFromFrame(s.Bg)),
		UnderlineColor: profile.Convert(colorFromFrame(s.UnderlineColor)),
		Underline:      uv.Underline(s.UnderlineStyle),
		Attrs:          uint8(s.Attrs),
	}
}

func colorFromFrame(c termframe.Color) ansi.Color {
	switch c.Kind {
	case termframe.ColorBasic:
		return ansi.BasicColor(c.Value)
	case termframe.ColorIndexed:
		return ansi.IndexedColor(c.Value)
	case termframe.ColorRGB:
		return ansi.RGBColor{
			R: uint8(c.Value >> 16),
			G: uint8(c.Value >> 8),
			B: uint8(c.Value),
		}
	default:
		return nil
	}
}

func renderSpaces(b *strings.Builder, n int) {
	for n > 0 {
		_ = b.WriteByte(' ')
		n--
	}
}

func renderResetForSpace(b *strings.Builder, pen *uv.Style, link *termframe.Link) {
	if pen != nil && !pen.IsZero() {
		b.WriteString(ansi.ResetStyle)
		*pen = uv.Style{}
	}
	if link != nil && !link.IsZero() {
		b.WriteString(ansi.ResetHyperlink())
		*link = termframe.Link{}
	}
}

func renderFinalizeLine(b *strings.Builder, pen *uv.Style, link *termframe.Link) {
	if link != nil && !link.IsZero() {
		b.WriteString(ansi.ResetHyperlink())
		*link = termframe.Link{}
	}
	if pen != nil && !pen.IsZero() {
		b.WriteString(ansi.ResetStyle)
		*pen = uv.Style{}
	}
}

func renderApplyStyle(b *strings.Builder, pen *uv.Style, next uv.Style) {
	if pen == nil {
		return
	}
	if next.IsZero() {
		if !pen.IsZero() {
			b.WriteString(ansi.ResetStyle)
			*pen = uv.Style{}
		}
		return
	}
	if next.Equal(pen) {
		return
	}
	b.WriteString(next.Diff(pen))
	*pen = next
}

func renderApplyLink(b *strings.Builder, link *termframe.Link, next termframe.Link) {
	if link == nil {
		return
	}
	if next == *link {
		return
	}
	if !link.IsZero() {
		b.WriteString(ansi.ResetHyperlink())
		*link = termframe.Link{}
	}
	if next.IsZero() {
		return
	}
	b.WriteString(ansi.SetHyperlink(next.URL, next.Params))
	*link = next
}
