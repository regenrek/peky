package tmuxctl

import (
	"context"
	"reflect"
	"testing"
)

func TestSessionSnapshotParses(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{
			name:   "tmux",
			args:   []string{"list-windows", "-t", "demo", "-F", "#{window_index}\t#{window_name}\t#{window_active}"},
			stdout: "0\tmain\t1\n1\tlogs\t0\n",
			exit:   0,
		},
		{
			name:   "tmux",
			args:   []string{"list-panes", "-t", "demo:0", "-F", "#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}"},
			stdout: "0\t1\t\tcmd0\n1\t0\tTitle\tcmd1\n",
			exit:   0,
		},
		{
			name:   "tmux",
			args:   []string{"list-panes", "-t", "demo:1", "-F", "#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}"},
			stdout: "0\t1\t\tcmd2\n",
			exit:   0,
		},
	}}
	client := &Client{bin: "tmux", run: runner.run}
	snap, err := client.SessionSnapshot(context.Background(), "demo")
	if err != nil {
		t.Fatalf("SessionSnapshot() error: %v", err)
	}
	if snap.Session != "demo" || len(snap.Windows) != 2 {
		t.Fatalf("SessionSnapshot() = %#v", snap)
	}
	if snap.Windows[0].Name != "main" || len(snap.Windows[0].Panes) != 2 {
		t.Fatalf("SessionSnapshot() windows = %#v", snap.Windows)
	}
	wantTitles := []string{"cmd0", "Title"}
	var gotTitles []string
	for _, pane := range snap.Windows[0].Panes {
		gotTitles = append(gotTitles, pane.Title)
	}
	if !reflect.DeepEqual(gotTitles, wantTitles) {
		t.Fatalf("SessionSnapshot() pane titles = %#v", gotTitles)
	}
	runner.assertDone()
}

func TestSessionSnapshotMissingSession(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"list-windows", "-t", "missing", "-F", "#{window_index}\t#{window_name}\t#{window_active}"},
		exit: 1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	snap, err := client.SessionSnapshot(context.Background(), "missing")
	if err != nil {
		t.Fatalf("SessionSnapshot() error: %v", err)
	}
	if snap.Session != "missing" || len(snap.Windows) != 0 {
		t.Fatalf("SessionSnapshot() = %#v", snap)
	}
	runner.assertDone()
}
