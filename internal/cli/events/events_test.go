package events

import (
	"testing"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestParseRFC3339(t *testing.T) {
	if _, err := parseRFC3339("x"); err == nil {
		t.Fatalf("expected error")
	}
	if got, err := parseRFC3339(" "); err != nil || !got.IsZero() {
		t.Fatalf("blank got=%v err=%v", got, err)
	}
	ts := time.Now().UTC().Truncate(time.Second)
	got, err := parseRFC3339(ts.Format(time.RFC3339))
	if err != nil || !got.Equal(ts) {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestEventFilter(t *testing.T) {
	f := eventFilter([]string{" focus ", "", "pane.updated"})
	if len(f) != 2 {
		t.Fatalf("len=%d", len(f))
	}
	if _, ok := f[sessiond.EventType("focus")]; !ok {
		t.Fatalf("missing focus")
	}
}

func TestCommandTimeoutPrefersFlag(t *testing.T) {
	cmd := &cli.Command{
		Name: "events",
		Flags: []cli.Flag{
			&cli.DurationFlag{Name: "timeout"},
		},
	}
	_ = cmd.Set("timeout", "2s")
	ctx := root.CommandContext{Cmd: cmd}
	if got := commandTimeout(ctx); got != 2*time.Second {
		t.Fatalf("timeout=%v", got)
	}
}
