package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/app"
)

type startOptions struct {
	layoutName   string
	session      string
	path         string
	showHelp     bool
	freshConfig  bool
	temporaryRun bool
}

func runStart(args []string) {
	opts := parseStartArgs(args)
	if opts.showHelp {
		fmt.Print(startHelpText)
		return
	}
	if opts.temporaryRun {
		opts.freshConfig = true
	}
	cleanup, err := applyRunEnv(runEnvOptions{
		freshConfig:  opts.freshConfig,
		temporaryRun: opts.temporaryRun,
	})
	if err != nil {
		fatal("%v", err)
	}
	defer cleanup()
	projectPath := strings.TrimSpace(opts.path)
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			fatal("cannot determine current directory: %v", err)
		}
	}
	autoStart := &app.AutoStartSpec{
		Session: opts.session,
		Path:    projectPath,
		Layout:  opts.layoutName,
		Focus:   true,
	}
	runMenuFn(autoStart)
}

func parseStartArgs(args []string) startOptions {
	opts := startOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--layout", "-l":
			if i+1 < len(args) {
				opts.layoutName = args[i+1]
				i++
			}
		case "--session", "-s":
			if i+1 < len(args) {
				opts.session = args[i+1]
				i++
			}
		case "--path", "-p":
			if i+1 < len(args) {
				opts.path = args[i+1]
				i++
			}
		case "--fresh-config":
			opts.freshConfig = true
		case "--temporary-run":
			opts.temporaryRun = true
		case "-h", "--help":
			opts.showHelp = true
		default:
			if !strings.HasPrefix(args[i], "-") && opts.layoutName == "" {
				opts.layoutName = args[i]
			}
		}
	}
	return opts
}
