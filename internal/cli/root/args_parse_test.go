package root

import (
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestPositionalArgsUsesNamedArguments(t *testing.T) {
	cmdSpec := spec.Command{
		ID: "nl.plan",
		Args: []spec.Arg{{
			Name:     "prompt",
			Required: true,
			Variadic: true,
		}},
	}
	cmd := &cli.Command{
		Name: "plan",
		Arguments: []cli.Argument{
			&cli.StringArgs{Name: "prompt", Min: 1, Max: -1},
		},
	}
	rargs := []string{"list", "sessions"}
	for _, a := range cmd.Arguments {
		var err error
		rargs, err = a.Parse(rargs)
		if err != nil {
			t.Fatalf("arg parse: %v", err)
		}
	}
	got := positionalArgs(cmdSpec, cmd)
	if len(got) != 2 || got[0] != "list" || got[1] != "sessions" {
		t.Fatalf("got=%v", got)
	}
}

func TestValidateArgsVariadicRequired(t *testing.T) {
	cmdSpec := spec.Command{
		ID: "nl.plan",
		Args: []spec.Arg{{
			Name:     "prompt",
			Required: true,
			Variadic: true,
		}},
	}
	cmd := &cli.Command{
		Name: "plan",
		Arguments: []cli.Argument{
			&cli.StringArgs{Name: "prompt", Min: 1, Max: -1},
		},
	}
	if err := validateArgs(cmdSpec, cmd); err == nil {
		t.Fatalf("expected error")
	}
	rargs := []string{"x"}
	for _, a := range cmd.Arguments {
		var err error
		rargs, err = a.Parse(rargs)
		if err != nil {
			t.Fatalf("arg parse: %v", err)
		}
	}
	if err := validateArgs(cmdSpec, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
