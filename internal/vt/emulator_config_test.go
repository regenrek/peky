package vt

import "testing"

func TestResolveParserDataSize(t *testing.T) {
	if got := resolveParserDataSize(""); got != defaultParserDataSize {
		t.Fatalf("default size = %d, want %d", got, defaultParserDataSize)
	}
	if got := resolveParserDataSize("not-a-number"); got != defaultParserDataSize {
		t.Fatalf("invalid size = %d, want %d", got, defaultParserDataSize)
	}
	if got := resolveParserDataSize("0"); got != defaultParserDataSize {
		t.Fatalf("zero size = %d, want %d", got, defaultParserDataSize)
	}
	if got := resolveParserDataSize("128"); got != 128 {
		t.Fatalf("size = %d, want 128", got)
	}
	if got := resolveParserDataSize("99999999"); got != maxParserDataSize {
		t.Fatalf("clamped size = %d, want %d", got, maxParserDataSize)
	}
}
