package pane

import (
	"strconv"

	"github.com/urfave/cli/v3"
)

func intFlagString(cmd *cli.Command, name string) string {
	if cmd == nil || name == "" {
		return ""
	}
	if cmd.IsSet(name) {
		return strconv.Itoa(cmd.Int(name))
	}
	return ""
}
