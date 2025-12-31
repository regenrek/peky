package dashboard

import "github.com/regenrek/peakypanes/internal/cli/root"

// Register registers the dashboard command.
func Register(reg *root.Registry) {
	reg.Register("dashboard", func(ctx root.CommandContext) error {
		return Run(ctx, nil)
	})
}
