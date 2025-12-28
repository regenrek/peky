package terminal

import "testing"

type testAttrs struct{}

func (testAttrs) Bold() bool   { return true }
func (testAttrs) Italic() bool { return false }

type badAttrs struct{}

func (badAttrs) Bold() string { return "nope" }

func TestAttrsBoolAndMaxInt(t *testing.T) {
	if !attrsBool(testAttrs{}, "Bold") {
		t.Fatalf("expected Bold true")
	}
	if attrsBool(testAttrs{}, "Unknown") {
		t.Fatalf("expected unknown method false")
	}
	if attrsBool(badAttrs{}, "Bold") {
		t.Fatalf("expected non-bool method false")
	}
	if attrsBool(nil, "Bold") {
		t.Fatalf("expected nil attrs false")
	}

	if maxInt(1, 2) != 2 || maxInt(3, -1) != 3 {
		t.Fatalf("unexpected maxInt result")
	}
}
