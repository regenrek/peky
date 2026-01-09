//go:build darwin

package terminal

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

func pidCwd(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lsof", "-a", "-p", pidString(pid), "-d", "cwd", "-Fn")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return "", false
	}
	path := parseLsofName(out.String())
	if path == "" {
		return "", false
	}
	return path, true
}

func parseLsofName(out string) string {
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if len(line) < 2 || line[0] != 'n' {
			continue
		}
		path := strings.TrimSpace(line[1:])
		if path != "" {
			return path
		}
	}
	return ""
}
