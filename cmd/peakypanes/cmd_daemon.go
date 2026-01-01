package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/runenv"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func runDaemon(args []string) {
	if len(args) > 0 && args[0] == "restart" {
		runDaemonRestart(args[1:])
		return
	}

	flags, err := parseDaemonFlags(args)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "peakypanes: %v\n", err); writeErr != nil {
			return
		}
		fmt.Print(daemonHelpText)
		return
	}
	if flags.showHelp {
		fmt.Print(daemonHelpText)
		return
	}
	if flags.pprofAddr != "" && !pprofSupported() {
		fatal("pprof requires a profiler build (rebuild with -tags profiler)")
	}

	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:                 version,
		StateDebounce:           sessiond.DefaultStateDebounce,
		HandleSignals:           true,
		SkipRestore:             runenv.FreshConfigEnabled(),
		DisableStatePersistence: runenv.FreshConfigEnabled(),
		PprofAddr:               flags.pprofAddr,
	})
	if err != nil {
		fatal("failed to create daemon: %v", err)
	}
	if err := daemon.Run(); err != nil {
		fatal("daemon failed: %v", err)
	}
}

const defaultPprofAddr = "127.0.0.1:6060"

type daemonFlags struct {
	showHelp  bool
	pprofAddr string
}

func parseDaemonFlags(args []string) (daemonFlags, error) {
	var flags daemonFlags
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "-h" || arg == "--help":
			flags.showHelp = true
		case arg == "--pprof":
			if flags.pprofAddr == "" {
				flags.pprofAddr = defaultPprofAddr
			}
		case strings.HasPrefix(arg, "--pprof-addr="):
			addr := strings.TrimSpace(strings.TrimPrefix(arg, "--pprof-addr="))
			if addr == "" {
				return flags, fmt.Errorf("pprof address is required")
			}
			flags.pprofAddr = addr
		case arg == "--pprof-addr":
			if i+1 >= len(args) {
				return flags, fmt.Errorf("pprof address is required")
			}
			i++
			addr := strings.TrimSpace(args[i])
			if addr == "" {
				return flags, fmt.Errorf("pprof address is required")
			}
			flags.pprofAddr = addr
		default:
			return flags, fmt.Errorf("unknown option %q", arg)
		}
	}
	return flags, nil
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
