package app

import "testing"

func TestQuickReplySummary(t *testing.T) {
	if got := quickReplySummary(""); got != "" {
		t.Fatalf("empty summary = %q", got)
	}
	if got := quickReplySummary("  hi  "); got != "hi" {
		t.Fatalf("trim summary = %q", got)
	}
	long := make([]byte, 130)
	for i := range long {
		long[i] = 'a'
	}
	got := quickReplySummary(string(long))
	if len(got) != 120 || got[len(got)-3:] != "..." {
		t.Fatalf("summary length = %d", len(got))
	}
}
