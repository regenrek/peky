package sessiond

import (
	"context"
	"errors"
)

// DaemonRunning reports whether a compatible daemon is reachable on the default socket path.
func DaemonRunning(ctx context.Context, version string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	socketPath, err := DefaultSocketPath()
	if err != nil {
		return false, err
	}
	if err := probeDaemon(ctx, socketPath, version); err != nil {
		if errors.Is(err, ErrDaemonProbeTimeout) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}
