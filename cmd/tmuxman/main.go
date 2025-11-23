package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kregenrek/tmuxman/internal/layout"
	"github.com/kregenrek/tmuxman/internal/tmuxctl"
	"github.com/kregenrek/tmuxman/internal/tui/create"
	"github.com/kregenrek/tmuxman/internal/tui/resume"
)

const (
	defaultSession = "grid"
)

type cliConfig struct {
	session string
	layout  layout.Grid
	dir     string
	attach  bool
}

func main() {
	var (
		defaultMode = flag.Bool("d", false, "use defaults (session=grid layout=2x2) without prompting")
		sessionFlag = flag.String("session", "", "tmux session name to create or attach")
		layoutFlag  = flag.String("layout", "", "grid layout like 2x2 or 2x3")
		dirFlag     = flag.String("C", "", "starting directory for all panes (defaults to current)")
		tmuxBin     = flag.String("tmux", "", "path to the tmux binary to execute")
		timeoutFlag = flag.Duration("timeout", 5*time.Second, "maximum time tmux commands may take")
		noAttach    = flag.Bool("no-attach", false, "create the session but do not attach/switch to it")
		resumeFlag  = flag.Bool("resume", false, "resume an existing tmux session (prompts if omitted)")
	)
	flag.Parse()

	cfg := cliConfig{attach: !*noAttach}

	client, err := tmuxctl.NewClient(*tmuxBin)
	if err != nil {
		exitWithErr(err)
	}

	if *resumeFlag {
		if err := resumeExisting(client, strings.TrimSpace(*sessionFlag)); err != nil {
			exitWithErr(err)
		}
		return
	}

	session := strings.TrimSpace(*sessionFlag)
	layoutSpec := strings.TrimSpace(*layoutFlag)
	startDir := strings.TrimSpace(*dirFlag)

	if *defaultMode {
		if session == "" {
			session = defaultSession
		}
		if layoutSpec == "" {
			layoutSpec = layout.Default.String()
		}
	}

	if session == "" || layoutSpec == "" {
		var err error
		session, layoutSpec, err = runInteractive(session, layoutSpec)
		if err != nil {
			exitWithErr(err)
		}
	}

	grid, err := layout.Parse(layoutSpec)
	if err != nil {
		exitWithErr(err)
	}

	cfg.session = session
	cfg.layout = grid
	cfg.dir = startDir

	ctx := context.Background()
	result, err := client.EnsureSession(ctx, tmuxctl.Options{
		Session:  cfg.session,
		Layout:   cfg.layout,
		StartDir: cfg.dir,
		Attach:   cfg.attach,
		Timeout:  *timeoutFlag,
	})
	if err != nil {
		exitWithErr(err)
	}

	if result.Created {
		fmt.Printf("Created tmux session %q with layout %s\n", cfg.session, cfg.layout)
	} else {
		fmt.Printf("Session %q already existed\n", cfg.session)
	}
	if cfg.attach && result.Attached {
		fmt.Println("Attached to session via tmux")
	}
}

func runInteractive(session, layoutSpec string) (string, string, error) {
	session = strings.TrimSpace(session)
	layoutSpec = strings.TrimSpace(layoutSpec)
	valSession, valLayout, err := create.Prompt(session, layoutSpec)
	if err != nil {
		if errors.Is(err, create.ErrAborted) {
			return "", "", fmt.Errorf("aborted by user")
		}
		return "", "", err
	}
	return valSession, valLayout, nil
}

func resumeExisting(client *tmuxctl.Client, requested string) error {
	session := strings.TrimSpace(requested)
	if session == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		sessions, err := client.ListSessions(ctx)
		cancel()
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			return errors.New("no tmux sessions are currently running to resume")
		}
		choice, err := resume.SelectSession(sessions)
		if err != nil {
			if errors.Is(err, resume.ErrAborted) {
				return fmt.Errorf("aborted by user")
			}
			return err
		}
		session = choice
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return client.AttachExisting(ctx, session)
}

func exitWithErr(err error) {
	fmt.Fprintf(os.Stderr, "tmuxman: %v\n", err)
	os.Exit(1)
}
