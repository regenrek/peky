package app

import "testing"

func TestEmptyStateMessage(t *testing.T) {
	m := newTestModel(t)
	if m.emptyStateMessage() == "" {
		t.Fatalf("emptyStateMessage() empty")
	}
}
