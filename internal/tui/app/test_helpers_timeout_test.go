package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/sessiond/testkit"
)

func TestWaitForSessionSnapshotTimeout(t *testing.T) {
	m := newTestModel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, err := testkit.WaitForSessionSnapshot(ctx, m.client, "missing-session")
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "missing-session") {
		t.Fatalf("expected error to include session name, got %q", err.Error())
	}
}
