package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"

	"github.com/regenrek/peakypanes/internal/devwatch"
)

func main() {
	var (
		watchFlag    = flag.String("watch", "cmd,internal,assets,go.mod,go.sum", "Comma-separated watch paths")
		extFlag      = flag.String("ext", ".go,.yml,.yaml,.json,.toml,.md", "Comma-separated file extensions for watched dirs")
		buildFlag    = flag.String("build", "go build -o ./peakypanes ./cmd/peakypanes", "Build command")
		runFlag      = flag.String("run", "./peakypanes", "Run command")
		intervalFlag = flag.Duration("interval", 300*time.Millisecond, "Polling interval")
		debounceFlag = flag.Duration("debounce", 200*time.Millisecond, "Debounce window")
		stopFlag     = flag.Duration("shutdown-timeout", 2*time.Second, "Shutdown timeout before kill")
	)
	flag.Parse()

	buildCmd, err := shellquote.Split(*buildFlag)
	if err != nil {
		log.Printf("devwatch: invalid build command: %v", err)
		os.Exit(1)
	}
	runCmd, err := shellquote.Split(*runFlag)
	if err != nil {
		log.Printf("devwatch: invalid run command: %v", err)
		os.Exit(1)
	}

	cfg := devwatch.Config{
		BuildCmd:        buildCmd,
		RunCmd:          append(runCmd, flag.Args()...),
		WatchPaths:      splitCSV(*watchFlag),
		Extensions:      splitCSV(*extFlag),
		Interval:        *intervalFlag,
		Debounce:        *debounceFlag,
		ShutdownTimeout: *stopFlag,
		Logger:          log.New(os.Stdout, "devwatch: ", log.LstdFlags),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := devwatch.Run(ctx, cfg); err != nil {
		log.Printf("devwatch: %v", err)
		os.Exit(1)
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
