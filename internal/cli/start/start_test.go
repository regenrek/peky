package start

import (
	"context"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

func TestRunStartJSONWithoutDaemonConnectErrors(t *testing.T) {
	workDir := t.TempDir()
	cmd := &cli.Command{
		Name: "start",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "layout"},
			&cli.StringFlag{Name: "session"},
			&cli.StringFlag{Name: "path"},
			&cli.BoolFlag{Name: "json"},
		},
	}
	_ = cmd.Set("json", "true")
	ctx := root.CommandContext{
		Context: context.Background(),
		Cmd:     cmd,
		Deps:    root.Dependencies{Version: "test", Connect: nil},
		JSON:    true,
		WorkDir: workDir,
	}
	err := runStart(ctx)
	if err == nil || !strings.Contains(err.Error(), "daemon connection not configured") {
		t.Fatalf("err=%v", err)
	}
}
