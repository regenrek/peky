package testkit

import (
	"context"
	"testing"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestWaitForSessionSnapshotValidatesInputs(t *testing.T) {
	if _, err := WaitForSessionSnapshot(context.Background(), nil, "s"); err == nil {
		t.Fatalf("expected error for nil client")
	}
	var c sessiond.Client
	if _, err := WaitForSessionSnapshot(context.Background(), &c, " "); err == nil {
		t.Fatalf("expected error for empty session name")
	}
}

func TestShouldCheckSnapshotOnEvent(t *testing.T) {
	if !shouldCheckSnapshotOnEvent(sessiond.EventSessionChanged) {
		t.Fatalf("expected session changed to trigger snapshot")
	}
	if shouldCheckSnapshotOnEvent(sessiond.EventType("other")) {
		t.Fatalf("expected other to not trigger snapshot")
	}
}
