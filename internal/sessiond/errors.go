package sessiond

import "errors"

var ErrDaemonProbeTimeout = errors.New("sessiond: daemon probe timed out")
