package help

import (
	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

// Register registers help handler.
func Register(reg *root.Registry) {
	reg.Register("help", runHelp)
}

func runHelp(ctx root.CommandContext) error {
	return cli.ShowRootCommandHelp(ctx.Cmd)
}
