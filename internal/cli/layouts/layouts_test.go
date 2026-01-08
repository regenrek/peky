package layouts

import (
	"bytes"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

func TestFormatLayoutSource(t *testing.T) {
	if got := formatLayoutSource("builtin"); !strings.Contains(got, "builtin") {
		t.Fatalf("builtin=%q", got)
	}
	if got := formatLayoutSource("x"); got != "x" {
		t.Fatalf("unknown=%q", got)
	}
}

func TestTruncateLayoutDescription(t *testing.T) {
	long := strings.Repeat("a", 80)
	got := truncateLayoutDescription(long)
	if len(got) != 50 || !strings.HasSuffix(got, "...") {
		t.Fatalf("got=%q len=%d", got, len(got))
	}
}

func TestRunExportJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	var out bytes.Buffer
	cmd := &cli.Command{
		Name: "layouts",
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "name"},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json"},
		},
		Writer:    &out,
		ErrWriter: &out,
	}
	_ = cmd.Set("json", "true")
	args := []string{"auto"}
	for _, a := range cmd.Arguments {
		var err error
		args, err = a.Parse(args)
		if err != nil {
			t.Fatalf("parse arg: %v", err)
		}
	}
	ctx := root.CommandContext{
		Cmd:    cmd,
		Deps:   root.Dependencies{Version: "test"},
		Out:    &out,
		ErrOut: &out,
		JSON:   true,
	}
	if err := runExport(ctx); err != nil {
		t.Fatalf("runExport error: %v", err)
	}
	if !strings.Contains(out.String(), "\"name\":\"auto\"") {
		t.Fatalf("out=%q", out.String())
	}
}
