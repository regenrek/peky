package dashboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"

	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/app"
	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

type programRunner interface {
	Run() (tea.Model, error)
	Send(tea.Msg)
}

type menuDeps struct {
	connect    func(ctx context.Context, version string) (*sessiond.Client, error)
	newModel   func(*sessiond.Client) (*app.Model, error)
	openInput  func() (*os.File, func(), error)
	newProgram func(tea.Model, ...tea.ProgramOption) programRunner
}

type resolvedMenuDeps struct {
	connect    func(ctx context.Context, version string) (*sessiond.Client, error)
	newModel   func(*sessiond.Client) (*app.Model, error)
	openInput  func() (*os.File, func(), error)
	newProgram func(tea.Model, ...tea.ProgramOption) programRunner
}

var (
	connectDefaultFn = sessiond.ConnectDefault
	newModelFn       = app.NewModel
	openTUIInputFn   = openTUIInput
	newProgramFn     = func(model tea.Model, opts ...tea.ProgramOption) programRunner {
		return tea.NewProgram(model, opts...)
	}
)

// Run launches the dashboard UI.
func Run(ctx root.CommandContext, autoStart *app.AutoStartSpec) error {
	return runMenuWith(ctx, autoStart, menuDeps{})
}

func resolveMenuDeps(deps menuDeps) resolvedMenuDeps {
	rd := resolvedMenuDeps(deps)
	if rd.connect == nil {
		rd.connect = connectDefaultFn
	}
	if rd.newModel == nil {
		rd.newModel = newModelFn
	}
	if rd.openInput == nil {
		rd.openInput = openTUIInputFn
	}
	if rd.newProgram == nil {
		rd.newProgram = newProgramFn
	}
	return rd
}

func runMenuWith(ctx root.CommandContext, autoStart *app.AutoStartSpec, deps menuDeps) error {
	rd := resolveMenuDeps(deps)

	connectTimeout := 20 * time.Second
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, connectTimeout)
	defer cancel()
	client, err := rd.connect(ctxTimeout, ctx.Deps.Version)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("failed to connect to daemon within %s (try `%s daemon` to run it in the foreground): %w", connectTimeout, ctx.Deps.AppName, err)
		}
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = client.Close() }()

	model, err := rd.newModel(client)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	if ctx.Out != nil {
		model.SetOSCWriter(ctx.Out)
	} else {
		model.SetOSCWriter(os.Stdout)
	}
	if autoStart != nil {
		model.SetAutoStart(*autoStart)
	}

	ttyIn, ttyCleanup, err := rd.openInput()
	if err != nil {
		return fmt.Errorf("cannot initialize TUI input: %w", err)
	}
	defer ttyCleanup()

	prevState, err := term.MakeRaw(ttyIn.Fd())
	if err != nil {
		return fmt.Errorf("cannot enter raw mode: %w", err)
	}
	defer func() { _ = term.Restore(ttyIn.Fd(), prevState) }()

	ttyOut := os.Stdout
	if term.IsTerminal(ttyOut.Fd()) {
		_, _ = io.WriteString(ttyOut, ansi.PushKittyKeyboard(ansi.KittyAllFlags))
		defer func() { _, _ = io.WriteString(ttyOut, ansi.PopKittyKeyboard(1)) }()
	}

	motionFilter := app.NewMouseMotionFilter()
	p := rd.newProgram(
		model,
		tea.WithAltScreen(),
		tea.WithInput(nil),
		tea.WithMouseAllMotion(),
		tea.WithFilter(motionFilter.Filter),
	)

	stream, err := tuiinput.Start(ctx.Context, p, ttyIn, os.Getenv("TERM"))
	if err != nil {
		return fmt.Errorf("cannot start input stream: %w", err)
	}
	defer stream.Stop()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
