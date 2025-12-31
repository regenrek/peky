package main

import "fmt"

type dashboardOptions struct {
	showHelp     bool
	freshConfig  bool
	temporaryRun bool
}

func runDashboardCommand(args []string) {
	opts := dashboardOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--fresh-config":
			opts.freshConfig = true
		case "--temporary-run":
			opts.temporaryRun = true
		case "-h", "--help":
			opts.showHelp = true
		}
	}

	if opts.showHelp {
		fmt.Print(dashboardHelpText)
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
	runMenuFn(nil)
}
