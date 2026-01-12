package version

import (
	"fmt"

	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/identity"
)

// Register registers version handler.
func Register(reg *root.Registry) {
	reg.Register("version", runVersion)
}

func runVersion(ctx root.CommandContext) error {
	_, err := fmt.Fprintf(ctx.Out, "%s %s\n", identity.CLIName, ctx.Deps.Version)
	return err
}
