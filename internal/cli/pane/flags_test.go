package pane

import (
	"context"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestIntFlagString(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		got := runIntFlag(t, []string{"test", "--index", "3"})
		if got != "3" {
			t.Fatalf("expected index 3, got %q", got)
		}
	})
	t.Run("zero", func(t *testing.T) {
		got := runIntFlag(t, []string{"test", "--index", "0"})
		if got != "0" {
			t.Fatalf("expected index 0, got %q", got)
		}
	})
	t.Run("unset", func(t *testing.T) {
		got := runIntFlag(t, []string{"test"})
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})
}

func runIntFlag(t *testing.T, args []string) string {
	t.Helper()
	var got string
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "index"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			got = intFlagString(cmd, "index")
			return nil
		},
	}
	if err := cmd.Run(context.Background(), args); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	return got
}
