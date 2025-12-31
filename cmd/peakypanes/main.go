package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/app"
	"github.com/regenrek/peakypanes/internal/cli/root"
)

var version = "dev"

func main() {
	deps := root.DefaultDependencies(version)
	runner, err := app.NewRunner(deps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "peakypanes: %v\n", err)
		os.Exit(1)
	}
	if err := runner.Run(context.Background(), os.Args); err != nil {
		if exitErr, ok := err.(cli.ExitCoder); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "peakypanes: %v\n", err)
		os.Exit(1)
	}
}
