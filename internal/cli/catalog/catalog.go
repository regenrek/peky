package catalog

import (
	"github.com/regenrek/peakypanes/internal/cli/clone"
	"github.com/regenrek/peakypanes/internal/cli/contextpack"
	"github.com/regenrek/peakypanes/internal/cli/daemon"
	"github.com/regenrek/peakypanes/internal/cli/dashboard"
	"github.com/regenrek/peakypanes/internal/cli/events"
	"github.com/regenrek/peakypanes/internal/cli/help"
	"github.com/regenrek/peakypanes/internal/cli/initcfg"
	"github.com/regenrek/peakypanes/internal/cli/layouts"
	"github.com/regenrek/peakypanes/internal/cli/pane"
	"github.com/regenrek/peakypanes/internal/cli/relay"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/session"
	"github.com/regenrek/peakypanes/internal/cli/start"
	"github.com/regenrek/peakypanes/internal/cli/version"
	"github.com/regenrek/peakypanes/internal/cli/workspace"
)

// RegisterAll registers all CLI commands except nl.
func RegisterAll(reg *root.Registry) {
	if reg == nil {
		return
	}
	dashboard.Register(reg)
	start.Register(reg)
	daemon.Register(reg)
	initcfg.Register(reg)
	layouts.Register(reg)
	clone.Register(reg)
	session.Register(reg)
	pane.Register(reg)
	relay.Register(reg)
	events.Register(reg)
	contextpack.Register(reg)
	workspace.Register(reg)
	version.Register(reg)
	help.Register(reg)
}
