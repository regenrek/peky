package relay

import (
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestDurationString(t *testing.T) {
	if got := durationString(0); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := durationString(1500 * time.Millisecond); got == "" {
		t.Fatalf("expected non-empty")
	}
}

func TestRelayFromInfoCopiesFields(t *testing.T) {
	info := sessiond.RelayInfo{
		ID:       "r1",
		FromPane: "p1",
		ToPanes:  []string{"p2"},
		Scope:    "all",
		Mode:     sessiond.RelayMode("mirror"),
		Status:   sessiond.RelayStatus("running"),
		Delay:    5 * time.Millisecond,
		TTL:      2 * time.Second,
		Stats: sessiond.RelayStats{
			Lines: 10,
			Bytes: 20,
		},
	}
	got := relayFromInfo(info)
	if got.ID != "r1" || got.FromPaneID != "p1" || got.Scope != "all" {
		t.Fatalf("got=%#v", got)
	}
	if got.Mode != "mirror" || got.Status != "running" {
		t.Fatalf("got=%#v", got)
	}
	if got.Stats != (output.RelayStats{Lines: 10, Bytes: 20}) {
		t.Fatalf("stats=%#v", got.Stats)
	}
}
