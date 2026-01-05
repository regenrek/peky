package sessiond

import (
	"errors"
	"net"
	"os"
)

var (
	ErrDaemonProbeTimeout    = errors.New("sessiond: daemon probe timed out")
	ErrClientClosed          = errors.New("sessiond: client closed")
	ErrConnectionUnavailable = errors.New("sessiond: connection unavailable")
	ErrResponseChannelClosed = errors.New("sessiond: response channel closed")
)

// IsConnectionError reports whether an error indicates the daemon connection is unavailable.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrClientClosed) || errors.Is(err, ErrConnectionUnavailable) || errors.Is(err, ErrResponseChannelClosed) {
		return true
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	if os.IsNotExist(err) {
		return true
	}
	return false
}
