package daemon

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

func TestRunStopWrapsError(t *testing.T) {
	sentinel := errors.New("stop failed")
	orig := stopDaemon
	stopDaemon = func(ctx context.Context, version string) error {
		return sentinel
	}
	t.Cleanup(func() { stopDaemon = orig })

	err := runStop(testCommandContext())
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func testCommandContext() root.CommandContext {
	return root.CommandContext{
		Context: context.Background(),
		Deps:    root.Dependencies{Version: "test"},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
}
