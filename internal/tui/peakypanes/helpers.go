package peakypanes

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/regenrek/peakypanes/internal/layout"
)

func validateSessionName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("Name cannot be empty")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("Name contains control characters")
		}
	}
	return nil
}

func validateProjectPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func nextSessionName(base string, existing []string) string {
	base = layout.SanitizeSessionName(base)
	if base == "" {
		base = "session"
	}
	used := make(map[string]struct{}, len(existing))
	for _, name := range existing {
		used[name] = struct{}{}
	}
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}
