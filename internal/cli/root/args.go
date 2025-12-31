package root

import (
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func buildArguments(args []spec.Arg) []cli.Argument {
	if len(args) == 0 {
		return nil
	}
	out := make([]cli.Argument, 0, len(args))
	for _, arg := range args {
		out = append(out, buildArgument(arg))
	}
	return out
}

func buildArgument(arg spec.Arg) cli.Argument {
	name := strings.TrimSpace(arg.Name)
	if arg.Variadic {
		min := 0
		if arg.Required {
			min = 1
		}
		return &cli.StringArgs{
			Name: name,
			Min:  min,
			Max:  -1,
		}
	}
	return &cli.StringArg{Name: name}
}
