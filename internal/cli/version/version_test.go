package version

import (
	"bytes"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

func TestRunVersionWrites(t *testing.T) {
	var out bytes.Buffer
	ctx := root.CommandContext{
		Deps: root.Dependencies{Version: "test", AppName: "peky"},
		Out:  &out,
		Cmd:  &cli.Command{Name: "version"},
	}
	if err := runVersion(ctx); err != nil {
		t.Fatalf("runVersion error: %v", err)
	}
	if !strings.Contains(out.String(), "peky test") {
		t.Fatalf("out=%q", out.String())
	}
}
