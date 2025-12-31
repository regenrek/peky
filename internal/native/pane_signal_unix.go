//go:build !windows

package native

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

func signalFromName(name string) (os.Signal, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return syscall.SIGTERM, nil
	}
	normalized := strings.ToUpper(strings.TrimPrefix(trimmed, "SIG"))
	switch normalized {
	case "INT":
		return syscall.SIGINT, nil
	case "TERM":
		return syscall.SIGTERM, nil
	case "KILL":
		return syscall.SIGKILL, nil
	case "HUP":
		return syscall.SIGHUP, nil
	case "QUIT":
		return syscall.SIGQUIT, nil
	default:
		return nil, fmt.Errorf("native: unknown signal %q", name)
	}
}
