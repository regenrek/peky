package peakypanes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"

	"github.com/regenrek/peakypanes/internal/tmuxctl"
)

type cmdSpec struct {
	name   string
	args   []string
	stdout string
	stderr string
	exit   int
}

type fakeRunner struct {
	t     *testing.T
	specs []cmdSpec
	idx   int
}

func (f *fakeRunner) run(ctx context.Context, name string, args ...string) *exec.Cmd {
	f.t.Helper()
	if f.idx >= len(f.specs) {
		f.t.Fatalf("unexpected command: %s %v", name, args)
	}
	spec := f.specs[f.idx]
	f.idx++
	if spec.name != name {
		f.t.Fatalf("command name = %q, want %q", name, spec.name)
	}
	if !reflect.DeepEqual(args, spec.args) {
		f.t.Fatalf("command args = %#v, want %#v", args, spec.args)
	}
	return helperCmd(ctx, spec.stdout, spec.stderr, spec.exit)
}

func (f *fakeRunner) assertDone() {
	if f.idx != len(f.specs) {
		f.t.Fatalf("not all commands consumed: %d of %d", f.idx, len(f.specs))
	}
}

func helperCmd(ctx context.Context, stdout, stderr string, exit int) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"PEAKYPANES_HELPER_STDOUT="+stdout,
		"PEAKYPANES_HELPER_STDERR="+stderr,
		"PEAKYPANES_HELPER_EXIT="+strconv.Itoa(exit),
	)
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	stdout := os.Getenv("PEAKYPANES_HELPER_STDOUT")
	stderr := os.Getenv("PEAKYPANES_HELPER_STDERR")
	exitCode := 0
	if raw := os.Getenv("PEAKYPANES_HELPER_EXIT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			exitCode = parsed
		}
	}
	if stdout != "" {
		_, _ = fmt.Fprint(os.Stdout, stdout)
	}
	if stderr != "" {
		_, _ = fmt.Fprint(os.Stderr, stderr)
	}
	os.Exit(exitCode)
}

func newTestModel(t *testing.T, specs []cmdSpec) (*Model, *fakeRunner) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	client, err := tmuxctl.NewClient("tmux")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	var runner *fakeRunner
	if specs != nil {
		runner = &fakeRunner{t: t, specs: specs}
		client.WithExec(runner.run)
	}

	model, err := NewModel(client)
	if err != nil {
		t.Fatalf("NewModel() error: %v", err)
	}
	model.width = 80
	model.height = 24
	model.state = StateDashboard
	return model, runner
}
