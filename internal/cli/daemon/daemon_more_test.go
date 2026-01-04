package daemon

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

func TestResolvePprofAddr(t *testing.T) {
	cmd := &cli.Command{
		Name: "daemon",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "pprof-addr"},
			&cli.BoolFlag{Name: "pprof"},
		},
	}
	ctx := root.CommandContext{Cmd: cmd}

	if err := cmd.Set("pprof", "true"); err != nil {
		t.Fatalf("cmd.Set(pprof) error: %v", err)
	}
	addr, err := resolvePprofAddr(ctx)
	if err != nil || addr != defaultPprofAddr {
		t.Fatalf("resolvePprofAddr(pprof) = %q err=%v", addr, err)
	}

	if err := cmd.Set("pprof-addr", " "); err != nil {
		t.Fatalf("cmd.Set(pprof-addr) error: %v", err)
	}
	if _, err := resolvePprofAddr(ctx); err == nil {
		t.Fatalf("expected error for empty pprof-addr")
	}

	if err := cmd.Set("pprof-addr", "127.0.0.1:9999"); err != nil {
		t.Fatalf("cmd.Set(pprof-addr) error: %v", err)
	}
	addr, err = resolvePprofAddr(ctx)
	if err != nil || addr != "127.0.0.1:9999" {
		t.Fatalf("resolvePprofAddr(addr) = %q err=%v", addr, err)
	}
}

func TestRunRestartWrapsError(t *testing.T) {
	sentinel := errors.New("restart failed")
	orig := restartDaemon
	restartDaemon = func(ctx context.Context, version string) error {
		return sentinel
	}
	t.Cleanup(func() { restartDaemon = orig })

	ctx := root.CommandContext{
		Context: context.Background(),
		Deps:    root.Dependencies{Version: "test"},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
		Cmd:     &cli.Command{Name: "daemon"},
	}
	err := runRestart(ctx)
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
