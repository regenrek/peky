package main

import (
	"os"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestMainUsesEntryRun(t *testing.T) {
	prev := osExit
	t.Cleanup(func() { osExit = prev })

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"peakypanes", "--version"}

	prevExiter := cli.OsExiter
	cli.OsExiter = func(int) {}
	t.Cleanup(func() { cli.OsExiter = prevExiter })

	version = "test"
	code := -1
	osExit = func(c int) { code = c }

	main()
	if code != 0 {
		t.Fatalf("exit code=%d", code)
	}
}
