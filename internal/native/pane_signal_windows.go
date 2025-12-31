//go:build windows

package native

import (
	"fmt"
	"os"
	"strings"
)

func signalFromName(name string) (os.Signal, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return os.Interrupt, nil
	}
	normalized := strings.ToUpper(strings.TrimPrefix(trimmed, "SIG"))
	switch normalized {
	case "INT":
		return os.Interrupt, nil
	case "KILL", "TERM":
		return os.Kill, nil
	default:
		return nil, fmt.Errorf("native: unknown signal %q", name)
	}
}
