package tmuxctl

import (
	"context"
	"reflect"
	"testing"
)

func TestSessionHasClientsNoServer(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"list-clients", "-t", "demo", "-F", "#{client_tty}"},
		stdout: "no server running",
		exit:   1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	has, err := client.SessionHasClients(context.Background(), "demo")
	if err != nil {
		t.Fatalf("SessionHasClients() error: %v", err)
	}
	if has {
		t.Fatalf("SessionHasClients() should be false")
	}
	runner.assertDone()
}

func TestHasClientOnTTY(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"list-clients", "-F", "#{client_tty}"},
		stdout: "/dev/ttys001\n/dev/ttys002\n",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	has, err := client.HasClientOnTTY(context.Background(), "/dev/ttys002")
	if err != nil {
		t.Fatalf("HasClientOnTTY() error: %v", err)
	}
	if !has {
		t.Fatalf("HasClientOnTTY() should be true")
	}
	runner.assertDone()
}

func TestHasClientOnTTYNoServer(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"list-clients", "-F", "#{client_tty}"},
		stdout: "no server running",
		exit:   1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	has, err := client.HasClientOnTTY(context.Background(), "/dev/ttys001")
	if err != nil {
		t.Fatalf("HasClientOnTTY() error: %v", err)
	}
	if has {
		t.Fatalf("HasClientOnTTY() should be false")
	}
	runner.assertDone()
}

func TestListSessionsInfoParses(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"list-sessions", "-F", "#{session_name}\t#{session_path}"},
		stdout: "app\t/Users/me/app\nempty\t#{session_path}\n",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	sessions, err := client.ListSessionsInfo(context.Background())
	if err != nil {
		t.Fatalf("ListSessionsInfo() error: %v", err)
	}
	want := []SessionInfo{{Name: "app", Path: "/Users/me/app"}, {Name: "empty", Path: ""}}
	if !reflect.DeepEqual(sessions, want) {
		t.Fatalf("ListSessionsInfo() = %#v", sessions)
	}
	runner.assertDone()
}

func TestListWindowsMissingSession(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"list-windows", "-t", "missing", "-F", "#{window_index}\t#{window_name}\t#{window_active}"},
		exit: 1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	windows, err := client.ListWindows(context.Background(), "missing")
	if err != nil {
		t.Fatalf("ListWindows() error: %v", err)
	}
	if len(windows) != 0 {
		t.Fatalf("ListWindows() = %#v", windows)
	}
	runner.assertDone()
}

func TestListPanesDetailedFallback(t *testing.T) {
	fullFormat := "#{pane_id}\t#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}\t#{pane_start_command}\t#{pane_pid}\t#{pane_left}\t#{pane_top}\t#{pane_width}\t#{pane_height}\t#{pane_dead}\t#{pane_dead_status}\t#{pane_last_active}"
	legacyFormat := "#{pane_id}\t#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}\t#{pane_start_command}\t#{pane_left}\t#{pane_top}\t#{pane_width}\t#{pane_height}\t#{pane_dead}\t#{pane_dead_status}\t#{pane_last_active}"
	basicFormat := "#{pane_id}\t#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}\t#{pane_left}\t#{pane_top}\t#{pane_width}\t#{pane_height}"

	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"list-panes", "-t", "demo:0", "-F", fullFormat}, exit: 1},
		{name: "tmux", args: []string{"list-panes", "-t", "demo:0", "-F", legacyFormat}, exit: 1},
		{name: "tmux", args: []string{"list-panes", "-t", "demo:0", "-F", basicFormat}, stdout: "%1\t0\t1\t\tcmd\t0\t0\t80\t24\n", exit: 0},
	}}
	client := &Client{bin: "tmux", run: runner.run}
	panes, err := client.ListPanesDetailed(context.Background(), "demo:0")
	if err != nil {
		t.Fatalf("ListPanesDetailed() error: %v", err)
	}
	if len(panes) != 1 || panes[0].Title != "cmd" {
		t.Fatalf("ListPanesDetailed() panes = %#v", panes)
	}
	runner.assertDone()
}

func TestCapturePaneLinesAlternateScreen(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"capture-pane", "-p", "-J", "-e", "-a", "-t", "demo:0.0", "-S", "0", "-E", "-"},
		stdout: "a\nb\nc\nd\n",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	lines, err := client.CapturePaneLines(context.Background(), "demo:0.0", 2)
	if err != nil {
		t.Fatalf("CapturePaneLines() error: %v", err)
	}
	if !reflect.DeepEqual(lines, []string{"c", "d"}) {
		t.Fatalf("CapturePaneLines() = %#v", lines)
	}
	runner.assertDone()
}

func TestCapturePaneLinesFallback(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"capture-pane", "-p", "-J", "-e", "-a", "-t", "demo:0.0", "-S", "0", "-E", "-"}, exit: 1},
		{name: "tmux", args: []string{"capture-pane", "-p", "-J", "-e", "-t", "demo:0.0", "-S", "-3", "-E", "-"}, stdout: "x\ny\nz\n", exit: 0},
	}}
	client := &Client{bin: "tmux", run: runner.run}
	lines, err := client.CapturePaneLines(context.Background(), "demo:0.0", 3)
	if err != nil {
		t.Fatalf("CapturePaneLines() error: %v", err)
	}
	if !reflect.DeepEqual(lines, []string{"x", "y", "z"}) {
		t.Fatalf("CapturePaneLines() = %#v", lines)
	}
	runner.assertDone()
}
