package vt

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestDamageTrackerBasic(t *testing.T) {
	var d DamageTracker
	d.Resize(4, 3)
	st := d.Consume()
	if !st.Full || st.Width != 4 || st.Height != 3 {
		t.Fatalf("state=%#v", st)
	}
	d.MarkRow(1)
	st = d.Consume()
	if st.Full || len(st.DirtyRows) != 1 || st.DirtyRows[0] != 1 {
		t.Fatalf("dirty rows=%v", st.DirtyRows)
	}
	d.MarkRect(uv.Rect(0, 0, 4, 2))
	st = d.Consume()
	if len(st.DirtyRows) != 2 || st.DirtyRows[0] != 0 || st.DirtyRows[1] != 1 {
		t.Fatalf("rect dirty=%v", st.DirtyRows)
	}
}

func TestDamageTrackerScroll(t *testing.T) {
	var d DamageTracker
	d.Resize(3, 3)
	_ = d.Consume()
	d.MarkRow(2)
	d.MarkScroll(1)
	st := d.Consume()
	if st.ScrollDy != 1 {
		t.Fatalf("scroll=%d", st.ScrollDy)
	}
	if len(st.DirtyRows) == 0 {
		t.Fatalf("expected dirty rows")
	}
	_ = d.Consume()
	d.MarkScroll(5)
	st = d.Consume()
	if !st.Full {
		t.Fatalf("expected full after out-of-bounds scroll")
	}
}
