package version

import (
	"fmt"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

// Register registers version handler.
func Register(reg *root.Registry) {
	reg.Register("version", runVersion)
}

func runVersion(ctx root.CommandContext) error {
	_, err := fmt.Fprintf(ctx.Out, "peakypanes %s\n", ctx.Deps.Version)
	return err
}
