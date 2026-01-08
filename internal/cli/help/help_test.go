package help

import (
	"bytes"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

func TestRunHelpWritesUsage(t *testing.T) {
	var out bytes.Buffer
	cmd := &cli.Command{Name: "peky", Writer: &out, ErrWriter: &out}
	ctx := root.CommandContext{Cmd: cmd, Out: &out, ErrOut: &out}
	if err := runHelp(ctx); err != nil {
		t.Fatalf("runHelp error: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("USAGE:")) {
		t.Fatalf("out=%q", out.String())
	}
}
