package app

import (
	"github.com/regenrek/peakypanes/internal/cli/catalog"
	"github.com/regenrek/peakypanes/internal/cli/nl"
	"github.com/regenrek/peakypanes/internal/cli/root"
)

func registerAll(reg *root.Registry) {
	if reg == nil {
		return
	}
	catalog.RegisterAll(reg)
	nl.Register(reg)
}
