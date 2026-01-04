package root

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
	"github.com/regenrek/peakypanes/internal/identity"
)

// Runner executes the CLI using the spec and registry.
type Runner struct {
	specDoc *spec.Spec
	deps    Dependencies
	app     *cli.Command
}

// NewRunner builds the CLI runner.
func NewRunner(specDoc *spec.Spec, deps Dependencies, reg *Registry) (*Runner, error) {
	app, err := BuildApp(specDoc, deps, reg)
	if err != nil {
		return nil, err
	}
	return &Runner{specDoc: specDoc, deps: deps, app: app}, nil
}

// Run executes the CLI with the given arguments.
func (r *Runner) Run(ctx context.Context, args []string) error {
	if r == nil || r.app == nil {
		return fmt.Errorf("runner is not initialized")
	}
	if r.specDoc != nil && r.app != nil {
		appName := identity.ResolveBinaryName(args)
		r.specDoc.App.Name = appName
		r.app.Name = appName
	}
	args = applyShorthand(r.specDoc, args)
	return r.app.Run(ctx, args)
}

func applyShorthand(specDoc *spec.Spec, args []string) []string {
	if specDoc == nil || len(args) == 0 {
		return args
	}
	if len(args) == 1 && strings.TrimSpace(specDoc.App.DefaultCommand) != "" {
		return []string{args[0], specDoc.App.DefaultCommand}
	}
	if !specDoc.App.AllowLayoutShorthand {
		return args
	}
	if len(args) == 2 && !strings.HasPrefix(args[1], "-") {
		if isTopLevelCommand(specDoc, args[1]) {
			return args
		}
		return []string{args[0], "start", "--layout", args[1]}
	}
	return args
}

func isTopLevelCommand(specDoc *spec.Spec, value string) bool {
	if specDoc == nil {
		return false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, cmd := range specDoc.Commands {
		if strings.EqualFold(cmd.Name, value) {
			return true
		}
		for _, alias := range cmd.Aliases {
			if strings.EqualFold(alias, value) {
				return true
			}
		}
	}
	return false
}
