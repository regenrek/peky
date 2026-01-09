package app

import "testing"

func TestNormalizeRuneRangeClampsAndOrders(t *testing.T) {
	tests := []struct {
		name      string
		start     int
		end       int
		max       int
		wantStart int
		wantEnd   int
	}{
		{name: "in_range", start: 1, end: 3, max: 5, wantStart: 1, wantEnd: 3},
		{name: "swap", start: 4, end: 2, max: 5, wantStart: 2, wantEnd: 4},
		{name: "clamp_low", start: -2, end: 2, max: 5, wantStart: 0, wantEnd: 2},
		{name: "clamp_high", start: 10, end: 2, max: 5, wantStart: 2, wantEnd: 5},
		{name: "clamp_both", start: -1, end: 99, max: 5, wantStart: 0, wantEnd: 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotStart, gotEnd := normalizeRuneRange(tc.start, tc.end, tc.max)
			if gotStart != tc.wantStart || gotEnd != tc.wantEnd {
				t.Fatalf("normalizeRuneRange(%d,%d,%d)=(%d,%d) want (%d,%d)", tc.start, tc.end, tc.max, gotStart, gotEnd, tc.wantStart, tc.wantEnd)
			}
		})
	}
}
