package root

import (
	"context"
	"io"
	"os"

	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Dependencies provides external services for CLI handlers.
type Dependencies struct {
	Version string
	AppName string
	WorkDir string

	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader

	Connect func(ctx context.Context, version string) (*sessiond.Client, error)
}

// DefaultDependencies returns dependencies wired to production services.
func DefaultDependencies(version string) Dependencies {
	return Dependencies{
		Version: version,
		AppName: identity.CLIName,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
		Connect: sessiond.ConnectDefault,
	}
}
