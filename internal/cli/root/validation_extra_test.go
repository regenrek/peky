package root

import (
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestValidateConstraintAtLeastOne(t *testing.T) {
	cmdSpec := spec.Command{
		Constraints: []spec.Constraint{{
			Type:   "at_least_one",
			Fields: []string{"foo", "bar"},
		}},
	}
	cmd := &cli.Command{
		Name:  "test",
		Flags: []cli.Flag{&cli.StringFlag{Name: "foo"}, &cli.StringFlag{Name: "bar"}},
	}
	if err := validateConstraints(cmdSpec, cmd); err == nil {
		t.Fatalf("expected error for missing fields")
	}
	if err := cmd.Set("foo", "ok"); err != nil {
		t.Fatalf("cmd.Set(foo) error: %v", err)
	}
	if err := validateConstraints(cmdSpec, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateConstraintRequiresAndExcludes(t *testing.T) {
	cmdSpec := spec.Command{
		Constraints: []spec.Constraint{
			{Type: "requires", Fields: []string{"foo", "bar"}},
			{Type: "excludes", Fields: []string{"baz", "qux"}},
		},
	}
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "foo"},
			&cli.StringFlag{Name: "bar"},
			&cli.StringFlag{Name: "baz"},
			&cli.StringFlag{Name: "qux"},
		},
	}
	if err := cmd.Set("foo", "1"); err != nil {
		t.Fatalf("cmd.Set(foo) error: %v", err)
	}
	if err := validateConstraints(cmdSpec, cmd); err == nil {
		t.Fatalf("expected requires error")
	}
	if err := cmd.Set("bar", "1"); err != nil {
		t.Fatalf("cmd.Set(bar) error: %v", err)
	}
	if err := validateConstraints(cmdSpec, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cmd.Set("baz", "1"); err != nil {
		t.Fatalf("cmd.Set(baz) error: %v", err)
	}
	if err := cmd.Set("qux", "1"); err != nil {
		t.Fatalf("cmd.Set(qux) error: %v", err)
	}
	if err := validateConstraints(cmdSpec, cmd); err == nil {
		t.Fatalf("expected excludes error")
	}
}

func TestFieldPresentArgs(t *testing.T) {
	cmdSpec := spec.Command{
		Args: []spec.Arg{
			{Name: "name"},
			{Name: "paths", Variadic: true},
		},
	}
	cmd := &cli.Command{
		Name: "test",
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "name"},
			&cli.StringArgs{Name: "paths", Min: 0, Max: -1},
		},
	}
	rargs := []string{"alice", "one", "two"}
	for _, arg := range cmd.Arguments {
		var err error
		rargs, err = arg.Parse(rargs)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
	}
	if !fieldPresent("name", cmdSpec, cmd) {
		t.Fatalf("expected name present")
	}
	if !fieldPresent("paths", cmdSpec, cmd) {
		t.Fatalf("expected paths present")
	}
}
