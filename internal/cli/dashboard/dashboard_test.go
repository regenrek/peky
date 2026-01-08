package dashboard

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestRunMenuWithConnectError(t *testing.T) {
	sentinel := errors.New("boom")
	ctx := root.CommandContext{Context: context.Background(), Deps: root.Dependencies{Version: "test"}}
	err := runMenuWith(ctx, nil, menuDeps{
		connect: func(ctx context.Context, version string) (*sessiond.Client, error) {
			return nil, sentinel
		},
	})
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("runMenuWith err=%v", err)
	}
}

func TestOpenTUIInputWithFallsBackToStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	t.Cleanup(func() { _ = w.Close() })

	f, cleanup, err := openTUIInputWith(
		func(string, int, os.FileMode) (*os.File, error) {
			return nil, errors.New("no tty")
		},
		func(*os.File) error { return nil },
		r,
	)
	if err != nil {
		t.Fatalf("openTUIInputWith error: %v", err)
	}
	if f != r {
		t.Fatalf("expected stdin fallback")
	}
	cleanup()
}
