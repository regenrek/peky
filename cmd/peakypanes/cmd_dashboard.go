package main

import "fmt"

type dashboardOptions struct {
	showHelp bool
}

func runDashboardCommand(args []string) {
	opts := dashboardOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			opts.showHelp = true
		}
	}

	if opts.showHelp {
		fmt.Print(dashboardHelpText)
		return
	}
	runMenuFn(nil)
}
