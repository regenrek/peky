package termframe

import "testing"

func TestFrameFromLines(t *testing.T) {
	f := FrameFromLines(4, 2, []string{"ab", "cd"})
	if f.Empty() || f.Cols != 4 || f.Rows != 2 {
		t.Fatalf("frame=%#v", f)
	}
	if c := f.CellAt(1, 0); c == nil || c.Content != "b" {
		t.Fatalf("cell=%#v", c)
	}
	empty := FrameFromLines(0, 2, []string{"x"})
	if !empty.Empty() {
		t.Fatalf("expected empty")
	}
}
