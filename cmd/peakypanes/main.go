package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/app"
)

var version = "dev"

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
	runMenuFn        = runMenu
	connectDefaultFn = sessiond.ConnectDefault
	newModelFn       = app.NewModel
	openTUIInputFn   = openTUIInput
	newProgramFn     = func(model tea.Model, opts ...tea.ProgramOption) programRunner {
		return tea.NewProgram(model, opts...)
	}

	exitFn    = os.Exit
	stderr    = os.Stderr
	openFile  = os.OpenFile
	stdinFile = os.Stdin
)

func main() {
	if len(os.Args) < 2 {
		runMenuFn(nil)
		return
	}

	switch os.Args[1] {
	case "dashboard", "ui":
		runDashboardCommand(os.Args[2:])
	case "open", "o", "start":
		runStart(os.Args[2:])
	case "daemon":
		runDaemon(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "layouts":
		runLayouts(os.Args[2:])
	case "clone", "c":
		runClone(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("peakypanes %s\n", version)
	case "help", "-h", "--help":
		fmt.Print(helpText)
	default:
		if !strings.HasPrefix(os.Args[1], "-") {
			runStart(os.Args[1:])
		} else {
			fmt.Print(helpText)
		}
	}
}

func runMenu(autoStart *app.AutoStartSpec) {
	if err := runMenuWith(autoStart, menuDeps{
		connect:    connectDefaultFn,
		newModel:   newModelFn,
		openInput:  openTUIInputFn,
		newProgram: newProgramFn,
	}); err != nil {
		fatal("%v", err)
	}
}

func runMenuWith(autoStart *app.AutoStartSpec, deps menuDeps) error {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := connect(ctx, version)
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

func openTUIInput() (*os.File, func(), error) {
	return openTUIInputWith(openFile, ensureBlocking, stdinFile)
}

func openTUIInputWith(
	openFileFn func(string, int, os.FileMode) (*os.File, error),
	ensureBlockingFn func(*os.File) error,
	stdin *os.File,
) (*os.File, func(), error) {
	if openFileFn == nil {
		openFileFn = os.OpenFile
	}
	if ensureBlockingFn == nil {
		ensureBlockingFn = ensureBlocking
	}
	if stdin == nil {
		stdin = os.Stdin
	}

	if f, err := openFileFn("/dev/tty", os.O_RDWR, 0); err == nil {
		if err := ensureBlockingFn(f); err != nil {
			_ = f.Close()
			return nil, func() {}, fmt.Errorf("configure /dev/tty: %w", err)
		}
		return f, func() { _ = f.Close() }, nil
	}
	if err := ensureBlockingFn(stdin); err != nil {
		return nil, func() {}, fmt.Errorf("stdin is not a usable TUI input: %w", err)
	}
	return stdin, func() {}, nil
}

func fatal(format string, args ...interface{}) {
	if _, err := fmt.Fprintf(stderr, "peakypanes: "+format+"\n", args...); err != nil {
		exitFn(1)
	}
	exitFn(1)
}
