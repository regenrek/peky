package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/kregenrek/tmuxman/internal/layout"
	"github.com/kregenrek/tmuxman/internal/tmuxctl"
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
	)
	flag.Parse()

	cfg := cliConfig{attach: !*noAttach}

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
		if err := runInteractive(&session, &layoutSpec, session == "", layoutSpec == ""); err != nil {
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

	client, err := tmuxctl.NewClient(*tmuxBin)
	if err != nil {
		exitWithErr(err)
	}

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

func runInteractive(session *string, layoutSpec *string, needsSession, needsLayout bool) error {
	if !needsSession && !needsLayout {
		return nil
	}

	var fields []huh.Field

	if needsSession {
		if strings.TrimSpace(*session) == "" {
			*session = defaultSession
		}
		fields = append(fields,
			huh.NewInput().Title("Session name").Value(session).Validate(func(v string) error {
				v = strings.TrimSpace(v)
				if v == "" {
					return errors.New("session name cannot be empty")
				}
				if strings.ContainsAny(v, " :") {
					return errors.New("session names cannot contain spaces or colons")
				}
				return nil
			}),
		)
	}

	if needsLayout {
		if strings.TrimSpace(*layoutSpec) == "" {
			*layoutSpec = layout.Default.String()
		}
		fields = append(fields,
			huh.NewInput().Title("Layout (rows x columns)").Description("Examples: 2x2, 2x3, 3x3").Value(layoutSpec).Validate(func(v string) error {
				_, err := layout.Parse(v)
				return err
			}),
		)
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return fmt.Errorf("aborted by user")
		}
		return err
	}
	return nil
}

func exitWithErr(err error) {
	fmt.Fprintf(os.Stderr, "tmuxman: %v\n", err)
	os.Exit(1)
}
