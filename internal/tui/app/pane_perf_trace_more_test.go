package app

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestPanePerfTraceEnabledPaths(t *testing.T) {
	orig := slog.Default()
	t.Cleanup(func() { slog.SetDefault(orig) })
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Setenv(perfTraceAllEnv, "1")

	m := newTestModelLite()
	now := time.Unix(100, 0)

	m.perfNotePaneUpdated("p1", 10, now)
	m.perfNotePaneQueued("p1", "reason", now)
	m.perfNotePaneQueuedBatch(map[string]struct{}{"p1": {}}, "batch")

	req := sessiond.PaneViewRequest{PaneID: "p1", Cols: 80, Rows: 24}
	m.perfNotePaneViewRequest(req, now.Add(300*time.Millisecond))
	m.perfNotePaneViewResponse(sessiond.PaneViewResponse{PaneID: "p1", UpdateSeq: 11}, now.Add(600*time.Millisecond))

	if ctx := m.panePerfContext("p1"); ctx == "" {
		t.Fatalf("expected perf context")
	}
}
