package main

import (
	"fmt"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func runDaemon(args []string) {
	showHelp := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			showHelp = true
		}
	}
	if showHelp {
		fmt.Print(daemonHelpText)
		return
	}

	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:       version,
		HandleSignals: true,
	})
	if err != nil {
		fatal("failed to create daemon: %v", err)
	}
	if err := daemon.Run(); err != nil {
		fatal("daemon failed: %v", err)
	}
}
