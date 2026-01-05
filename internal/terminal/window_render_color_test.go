package terminal

import (
	"image/color"
	"testing"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/regenrek/peakypanes/internal/termframe"
)

func TestColorFromColor(t *testing.T) {
	tests := []struct {
		name  string
		input color.Color
		want  termframe.Color
	}{
		{name: "nil", input: nil, want: termframe.Color{}},
		{name: "basic-color", input: xansi.BasicColor(4), want: termframe.Color{Kind: termframe.ColorBasic, Value: 4}},
		{name: "indexed-color", input: xansi.IndexedColor(196), want: termframe.Color{Kind: termframe.ColorIndexed, Value: 196}},
		{name: "rgb-color-112233", input: xansi.RGBColor{R: 0x11, G: 0x22, B: 0x33}, want: termframe.Color{Kind: termframe.ColorRGB, Value: 0x112233}},
		{name: "rgb-color-123456", input: xansi.RGBColor{R: 0x12, G: 0x34, B: 0x56}, want: termframe.Color{Kind: termframe.ColorRGB, Value: 0x123456}},
		{name: "hex-color", input: xansi.HexColor("#aabbcc"), want: termframe.Color{Kind: termframe.ColorRGB, Value: 0xaabbcc}},
		{name: "alpha-zero", input: color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0x00}, want: termframe.Color{}},
		{name: "rgba", input: color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}, want: termframe.Color{Kind: termframe.ColorRGB, Value: 0x123456}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := termframe.ColorFromColor(tc.input)
			if got != tc.want {
				t.Fatalf("ColorFromColor(%v) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}
