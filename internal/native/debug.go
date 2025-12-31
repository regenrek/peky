package native

import (
	"os"
	"strings"
	"sync"
)

var (
	perfDebugOnce sync.Once
	perfDebugOn   bool
)

func perfDebugEnabled() bool {
	perfDebugOnce.Do(func() {
		value := strings.TrimSpace(os.Getenv("PEAKYPANES_PERF_DEBUG"))
		if value == "" {
			perfDebugOn = false
			return
		}
		switch strings.ToLower(value) {
		case "0", "false", "no", "off":
			perfDebugOn = false
		default:
			perfDebugOn = true
		}
	})
	return perfDebugOn
}
