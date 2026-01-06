package root

import (
	"context"
	"io"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

// CommandContext wraps a command invocation.
type CommandContext struct {
	Context context.Context
	Args    []string
	Spec    spec.Command
	Cmd     *cli.Command
	Deps    Dependencies
	JSON    bool
	Out     io.Writer
	ErrOut  io.Writer
	Stdin   io.Reader
	WorkDir string
}
