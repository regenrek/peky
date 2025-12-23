package tmuxctl

import (
	"os"
	"os/exec"
	"strings"
)

// CurrentTTY returns the current terminal device path, if any.
func CurrentTTY() string {
	if tty := readLinkTTY("/dev/fd/0"); tty != "" {
		return tty
	}
	if tty := readLinkTTY("/proc/self/fd/0"); tty != "" {
		return tty
	}
	out, err := exec.Command("tty").Output()
	if err != nil {
		return ""
	}
	tty := strings.TrimSpace(string(out))
	if tty == "" || strings.Contains(strings.ToLower(tty), "not a tty") {
		return ""
	}
	if strings.HasPrefix(tty, "/dev/") {
		return tty
	}
	return ""
}

func readLinkTTY(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	target = strings.TrimSpace(target)
	if target == "" || !strings.HasPrefix(target, "/dev/") {
		return ""
	}
	return target
}
