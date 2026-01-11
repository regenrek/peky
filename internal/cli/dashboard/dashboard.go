package dashboard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/cancelreader"

	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/app"
)

type programRunner interface {
	Run() (tea.Model, error)
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

func buildTUIInput(openInput func() (*os.File, func(), error)) (cancelreader.File, func(), error) {
	inputFile, cleanup, err := openInput()
	if err != nil {
		return nil, func() {}, err
	}
	cleanupFns := []func(){cleanup}
	closeAll := func() {
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			cleanupFns[i]()
		}
	}

	var input cancelreader.File = inputFile
	if shouldTraceTUIInput() {
		traced, traceCleanup, err := newTracedTUIInput(input)
		if err != nil {
			closeAll()
			return nil, func() {}, err
		}
		cleanupFns = append(cleanupFns, traceCleanup)
		input = traced
	}

	input = newRepairedTUIInput(input)
	if shouldTraceTUIInputRepaired() {
		traced, traceCleanup, err := newTracedTUIInputRepaired(input)
		if err != nil {
			closeAll()
			return nil, func() {}, err
		}
		cleanupFns = append(cleanupFns, traceCleanup)
		input = traced
	}

	return input, closeAll, nil
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

	input, cleanup, err := buildTUIInput(rd.openInput)
	if err != nil {
		return fmt.Errorf("cannot initialize TUI input: %w", err)
	}
	defer cleanup()

	motionFilter := app.NewMouseMotionFilter()
	p := rd.newProgram(
		model,
		tea.WithAltScreen(),
		tea.WithInput(input),
		tea.WithMouseAllMotion(),
		tea.WithFilter(motionFilter.Filter),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
