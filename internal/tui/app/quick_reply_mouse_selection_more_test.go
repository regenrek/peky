package app

import "testing"

func TestRuneIndexAtQuickReplyXWideRunes(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.quickReplyInput.SetValue("a界b")

	inputRect, ok := m.quickReplyInputBounds()
	if !ok || inputRect.Empty() {
		t.Fatalf("quick reply input rect unavailable")
	}

	tests := []struct {
		name string
		x    int
		want int
	}{
		{name: "left_of_input", x: inputRect.X - 10, want: 0},
		{name: "at_start", x: inputRect.X, want: 0},
		// After 1 column: inside wide rune => idx 1.
		{name: "col_1", x: inputRect.X + 1, want: 1},
		{name: "col_2", x: inputRect.X + 2, want: 1},
		// After 3 columns: past wide rune => idx 2.
		{name: "col_3", x: inputRect.X + 3, want: 2},
		// Beyond end clamps to len(runes)=3.
		{name: "beyond_end", x: inputRect.X + 999, want: 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := m.runeIndexAtQuickReplyX(tc.x)
			if !ok {
				t.Fatalf("expected ok")
			}
			if got != tc.want {
				t.Fatalf("runeIndexAtQuickReplyX(%d)=%d want %d", tc.x, got, tc.want)
			}
		})
	}
}
