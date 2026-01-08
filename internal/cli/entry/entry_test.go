package entry

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestRunVersionFlagExitsZero(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	prevExiter := cli.OsExiter
	prevErrWriter := cli.ErrWriter
	cli.OsExiter = func(int) {}
	cli.ErrWriter = io.Discard
	t.Cleanup(func() {
		cli.OsExiter = prevExiter
		cli.ErrWriter = prevErrWriter
	})

	var out bytes.Buffer
	prevStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = prevStdout })
	t.Cleanup(func() { _ = r.Close() })
	t.Cleanup(func() { _ = w.Close() })

	exit := Run([]string{"peky", "--version"}, "test")
	_ = w.Close()
	_, _ = io.Copy(&out, r)
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if !strings.Contains(out.String(), "peky test") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestRunVersionCommandWrites(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	var out bytes.Buffer
	prevStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = prevStdout })
	t.Cleanup(func() { _ = r.Close() })
	t.Cleanup(func() { _ = w.Close() })

	exit := Run([]string{"peky", "version"}, "test")
	_ = w.Close()
	_, _ = io.Copy(&out, r)
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if !strings.Contains(out.String(), "peky test") {
		t.Fatalf("stdout=%q", out.String())
	}
}
