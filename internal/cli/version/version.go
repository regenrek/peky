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
	appName := identity.NormalizeCLIName(ctx.Deps.AppName)
	_, err := fmt.Fprintf(ctx.Out, "%s %s\n", appName, ctx.Deps.Version)
	return err
}
