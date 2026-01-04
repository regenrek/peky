package root

import (
	"bytes"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestBuildArguments(t *testing.T) {
	args := buildArguments([]spec.Arg{
		{Name: "name"},
		{Name: "paths", Variadic: true, Required: true},
	})
	if len(args) != 2 {
		t.Fatalf("buildArguments() len=%d", len(args))
	}
	if _, ok := args[0].(*cli.StringArg); !ok {
		t.Fatalf("expected StringArg for first arg")
	}
	variadic, ok := args[1].(*cli.StringArgs)
	if !ok {
		t.Fatalf("expected StringArgs for variadic arg")
	}
	if variadic.Min != 1 || variadic.Max != -1 {
		t.Fatalf("variadic bounds = %d..%d", variadic.Min, variadic.Max)
	}
}

func TestPromptConfirm(t *testing.T) {
	var out bytes.Buffer
	ok, err := PromptConfirm(strings.NewReader("y\n"), &out, "Confirm")
	if err != nil || !ok {
		t.Fatalf("PromptConfirm(y) ok=%v err=%v", ok, err)
	}
	ok, err = PromptConfirm(strings.NewReader("n\n"), nil, "Confirm")
	if err != nil || ok {
		t.Fatalf("PromptConfirm(n) ok=%v err=%v", ok, err)
	}
	if !strings.Contains(out.String(), "Confirm [y/N]:") {
		t.Fatalf("PromptConfirm output = %q", out.String())
	}
}

func TestRequireAck(t *testing.T) {
	var out bytes.Buffer
	ok, err := RequireAck(strings.NewReader("ack\n"), &out, "")
	if err != nil || !ok {
		t.Fatalf("RequireAck(default) ok=%v err=%v", ok, err)
	}
	ok, err = RequireAck(strings.NewReader("NOPE\n"), nil, "token")
	if err != nil || ok {
		t.Fatalf("RequireAck(mismatch) ok=%v err=%v", ok, err)
	}
	if !strings.Contains(out.String(), "Type \"ack\" to confirm:") {
		t.Fatalf("RequireAck output = %q", out.String())
	}
}

func TestValidateConstraintExactOne(t *testing.T) {
	cmdSpec := spec.Command{
		Constraints: []spec.Constraint{{
			Type:   "exactly_one",
			Fields: []string{"foo", "bar"},
		}},
	}
	cmd := &cli.Command{
		Name:  "test",
		Flags: []cli.Flag{&cli.StringFlag{Name: "foo"}, &cli.StringFlag{Name: "bar"}},
	}
	if err := cmd.Set("foo", "value"); err != nil {
		t.Fatalf("cmd.Set(foo) error: %v", err)
	}
	if err := validateConstraints(cmdSpec, cmd); err != nil {
		t.Fatalf("validateConstraints() error: %v", err)
	}
	if err := cmd.Set("bar", "value"); err != nil {
		t.Fatalf("cmd.Set(bar) error: %v", err)
	}
	if err := validateConstraints(cmdSpec, cmd); err == nil {
		t.Fatalf("expected error for multiple fields")
	}
}
