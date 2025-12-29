package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func runDaemon(args []string) {
	if len(args) > 0 && args[0] == "restart" {
		runDaemonRestart(args[1:])
		return
	}

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
	if len(args) > 0 {
		fmt.Print(daemonHelpText)
		return
	}

	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:       version,
		StateDebounce: sessiond.DefaultStateDebounce,
		HandleSignals: true,
	})
	if err != nil {
		fatal("failed to create daemon: %v", err)
	}
	if err := daemon.Run(); err != nil {
		fatal("daemon failed: %v", err)
	}
}

func runDaemonRestart(args []string) {
	showHelp := false
	assumeYes := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			showHelp = true
		case "-y", "--yes":
			assumeYes = true
		}
	}
	if showHelp {
		fmt.Print(daemonRestartHelpText)
		return
	}

	if _, err := fmt.Fprintln(stderr, "Restarting the daemon will disconnect clients; sessions will be restored."); err != nil {
		fatal("failed to write restart warning: %v", err)
	}
	if !assumeYes && !confirmPrompt("Restart daemon? [y/N]: ") {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := sessiond.RestartDaemon(ctx, version); err != nil {
		fatal("failed to restart daemon: %v", err)
	}
	if _, err := fmt.Fprintln(stderr, "Daemon restarted."); err != nil {
		fatal("failed to write restart confirmation: %v", err)
	}
}

func confirmPrompt(prompt string) bool {
	if _, err := fmt.Fprint(stderr, prompt); err != nil {
		return false
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	switch strings.ToLower(line) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
