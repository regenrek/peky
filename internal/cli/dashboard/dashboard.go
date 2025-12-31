package dashboard

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

func runMenuWith(ctx root.CommandContext, autoStart *app.AutoStartSpec, deps menuDeps) error {
	connect := deps.connect
	if connect == nil {
		connect = connectDefaultFn
	}
	newModel := deps.newModel
	if newModel == nil {
		newModel = newModelFn
	}
	openInput := deps.openInput
	if openInput == nil {
		openInput = openTUIInputFn
	}
	newProgram := deps.newProgram
	if newProgram == nil {
		newProgram = newProgramFn
	}

	ctxTimeout, cancel := context.WithTimeout(ctx.Context, 10*time.Second)
	defer cancel()
	client, err := connect(ctxTimeout, ctx.Deps.Version)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = client.Close() }()

	model, err := newModel(client)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	if autoStart != nil {
		model.SetAutoStart(*autoStart)
	}

	input, cleanup, err := openInput()
	if err != nil {
		return fmt.Errorf("cannot initialize TUI input: %w", err)
	}
	defer cleanup()

	motionFilter := app.NewMouseMotionFilter()
	p := newProgram(
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
