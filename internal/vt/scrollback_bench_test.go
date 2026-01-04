package vt

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/regenrek/peakypanes/internal/limits"
)

func BenchmarkScrollbackPushLine(b *testing.B) {
	widths := []int{80, 200}
	for _, width := range widths {
		b.Run("W"+itoa(width), func(b *testing.B) {
			sb := NewScrollback(limits.TerminalScrollbackMaxBytesMax)
			sb.SetWrapWidth(width)
			line := makeLine(width)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sb.PushLineWithWrap(line, false)
			}
		})
	}
}

func BenchmarkScrollbackReflow(b *testing.B) {
	sb := NewScrollback(limits.TerminalScrollbackMaxBytesMax)
	sb.SetWrapWidth(80)
	line := makeLine(80)
	for i := 0; i < 2000; i++ {
		sb.PushLineWithWrap(line, false)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.SetWrapWidth(120)
		sb.SetWrapWidth(80)
	}
}

func makeLine(width int) []uv.Cell {
	if width < 1 {
		width = 1
	}
	out := make([]uv.Cell, width)
	for i := 0; i < width; i++ {
		out[i] = uv.Cell{Content: "x", Width: 1}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(buf[i:])
}
