package app

import "testing"

func TestAutoSplitVertical(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
		want   bool
	}{
		{name: "wide", width: 120, height: 80, want: false},
		{name: "tall", width: 80, height: 120, want: true},
		{name: "square", width: 100, height: 100, want: false},
		{name: "zero width", width: 0, height: 50, want: false},
		{name: "zero height", width: 50, height: 0, want: false},
	}

	for _, tt := range cases {
		if got := autoSplitVertical(tt.width, tt.height); got != tt.want {
			t.Fatalf("%s: autoSplitVertical(%d, %d) = %v, want %v", tt.name, tt.width, tt.height, got, tt.want)
		}
	}
}
