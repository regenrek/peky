package logging

import "strings"

type Mode uint8

const (
	ModeCLI Mode = iota + 1
	ModeDaemon
)

func ModeFromArgs(args []string) Mode {
	if len(args) < 2 {
		return ModeCLI
	}
	cmd := strings.TrimSpace(args[1])
	if cmd == "" {
		return ModeCLI
	}
	cmd = strings.ToLower(cmd)
	if cmd == "daemon" || strings.HasPrefix(cmd, "daemon.") {
		return ModeDaemon
	}
	return ModeCLI
}

func (m Mode) String() string {
	switch m {
	case ModeDaemon:
		return "daemon"
	default:
		return "cli"
	}
}
