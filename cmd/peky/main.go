package main

import (
	"os"

	"github.com/regenrek/peakypanes/internal/cli/entry"
)

var version = "dev"
var osExit = os.Exit

func main() {
	osExit(entry.Run(os.Args, version))
}
