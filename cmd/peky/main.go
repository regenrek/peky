package main

import (
	"os"

	"github.com/regenrek/peakypanes/internal/cli/entry"
)

var version = "dev"

func main() {
	os.Exit(entry.Run(os.Args, version))
}
