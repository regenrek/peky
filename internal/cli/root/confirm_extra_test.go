package root

import (
	"bytes"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestConfirmIfNeeded(t *testing.T) {
	cmdSpec := spec.Command{ID: "cmd", SideEffects: true}
	cmd := &cli.Command{
		Name:  "cmd",
		Flags: []cli.Flag{&cli.BoolFlag{Name: "yes"}},
	}
	ctx := CommandContext{
		Spec:   cmdSpec,
		Stdin:  strings.NewReader("n\n"),
		ErrOut: &bytes.Buffer{},
	}
	if err := confirmIfNeeded(ctx, cmd); err == nil {
		t.Fatalf("expected confirmation error")
	}
	if err := cmd.Set("yes", "true"); err != nil {
		t.Fatalf("cmd.Set yes error: %v", err)
	}
	if err := confirmIfNeeded(ctx, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
