package terminal

import (
	"image/color"
	"testing"

	xansi "github.com/charmbracelet/x/ansi"
)

func TestColorToStyleToken(t *testing.T) {
	tests := []struct {
		name  string
		input color.Color
		want  string
	}{
		{name: "nil", input: nil, want: ""},
		{name: "basic-color", input: xansi.BasicColor(4), want: "4"},
		{name: "indexed-color", input: xansi.IndexedColor(196), want: "196"},
		{name: "true-color", input: xansi.TrueColor(0x112233), want: "#112233"},
		{name: "rgb-color", input: xansi.RGBColor{R: 0x12, G: 0x34, B: 0x56}, want: "#123456"},
		{name: "hex-color", input: xansi.HexColor("#aabbcc"), want: "#aabbcc"},
		{name: "alpha-zero", input: color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0x00}, want: ""},
		{name: "rgba", input: color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}, want: "#123456"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := colorToStyleToken(tc.input)
			if got != tc.want {
				t.Fatalf("colorToStyleToken(%v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
