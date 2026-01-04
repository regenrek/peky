package root

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/cli/spec"
	"github.com/urfave/cli/v3"
)

func TestBuildAppErrors(t *testing.T) {
	reg := NewRegistry()
	if _, err := BuildApp(nil, Dependencies{}, reg); err == nil {
		t.Fatalf("expected error for nil spec")
	}
	if _, err := BuildApp(&spec.Spec{}, Dependencies{}, nil); err == nil {
		t.Fatalf("expected error for nil registry")
	}
}

func TestBuildAppAndRunDefault(t *testing.T) {
	specDoc := &spec.Spec{
		App: spec.AppSpec{Name: "pp", DefaultCommand: "cmd"},
		Commands: []spec.Command{
			{Name: "cmd", ID: "cmd", Summary: "do"},
		},
	}
	reg := NewRegistry()
	called := false
	reg.Register("cmd", func(ctx CommandContext) error {
		called = true
		return nil
	})
	app, err := BuildApp(specDoc, Dependencies{}, reg)
	if err != nil {
		t.Fatalf("BuildApp() error: %v", err)
	}
	if err := app.Run(context.Background(), []string{"pp"}); err != nil {
		t.Fatalf("app.Run() error: %v", err)
	}
	if !called {
		t.Fatalf("expected handler called")
	}
}

func TestRunHandlerJSONUnsupported(t *testing.T) {
	specDoc := &spec.Spec{
		App:         spec.AppSpec{Name: "pp"},
		GlobalFlags: []spec.Flag{{Name: "json", Type: "bool"}},
		Commands: []spec.Command{
			{Name: "cmd", ID: "cmd", Summary: "do"},
		},
	}
	reg := NewRegistry()
	reg.Register("cmd", func(ctx CommandContext) error { return nil })
	app, err := BuildApp(specDoc, Dependencies{}, reg)
	if err != nil {
		t.Fatalf("BuildApp() error: %v", err)
	}
	if err := app.Run(context.Background(), []string{"pp", "cmd", "--json"}); err == nil {
		t.Fatalf("expected error for unsupported json")
	}
}

func TestRunHandlerJSONErrorResponse(t *testing.T) {
	specDoc := &spec.Spec{
		App:         spec.AppSpec{Name: "pp"},
		GlobalFlags: []spec.Flag{{Name: "json", Type: "bool"}},
		Commands: []spec.Command{
			{Name: "cmd", ID: "cmd", Summary: "do", JSON: &spec.JSONSpec{Supported: true}},
		},
	}
	var out bytes.Buffer
	reg := NewRegistry()
	reg.Register("cmd", func(ctx CommandContext) error {
		return errors.New("boom")
	})
	app, err := BuildApp(specDoc, Dependencies{Stdout: &out, Stderr: &out}, reg)
	if err != nil {
		t.Fatalf("BuildApp() error: %v", err)
	}
	app.ExitErrHandler = func(ctx context.Context, cmd *cli.Command, err error) {}
	if err := app.Run(context.Background(), []string{"pp", "cmd", "--json"}); err == nil {
		t.Fatalf("expected error from handler")
	}
	if !strings.Contains(out.String(), "command_failed") {
		t.Fatalf("expected json error output, got %q", out.String())
	}
}

func TestBuildAppVersionFlag(t *testing.T) {
	specDoc := &spec.Spec{
		App: spec.AppSpec{Name: "pp", DefaultCommand: "cmd"},
		GlobalFlags: []spec.Flag{
			{Name: "version", Aliases: []string{"v"}, Type: "bool"},
		},
		Commands: []spec.Command{
			{Name: "cmd", ID: "cmd", Summary: "do"},
			{Name: "version", ID: "version", Summary: "show"},
		},
	}
	reg := NewRegistry()
	called := false
	reg.Register("cmd", func(ctx CommandContext) error {
		called = true
		return nil
	})
	reg.Register("version", func(ctx CommandContext) error { return nil })
	var out bytes.Buffer
	app, err := BuildApp(specDoc, Dependencies{Version: "1.2.3", Stdout: &out, Stderr: &out}, reg)
	if err != nil {
		t.Fatalf("BuildApp() error: %v", err)
	}
	app.ExitErrHandler = func(ctx context.Context, cmd *cli.Command, err error) {}

	err = app.Run(context.Background(), []string{"pp", "--version"})
	if err == nil {
		t.Fatalf("expected exit error for --version")
	}
	exitErr, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("expected exit coder, got %T", err)
	}
	if exitErr.ExitCode() != 0 {
		t.Fatalf("expected exit code 0, got %d", exitErr.ExitCode())
	}
	if !strings.Contains(out.String(), "pp 1.2.3") {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if called {
		t.Fatalf("expected default command skipped")
	}
}
