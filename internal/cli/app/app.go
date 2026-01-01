package app

import (
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/spec"
)

// NewRunner builds the CLI runner from the embedded spec.
func NewRunner(deps root.Dependencies) (*root.Runner, error) {
	specDoc, err := spec.LoadDefault()
	if err != nil {
		return nil, err
	}
	reg := root.NewRegistry()
	registerAll(reg)
	return root.NewRunner(specDoc, deps, reg)
}
