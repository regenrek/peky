package zellijctl

import (
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestBuildLayoutFromWindows(t *testing.T) {
	cfg := &layout.LayoutConfig{
		Name: "dev",
		Windows: []layout.WindowDef{
			{
				Name: "editor",
				Panes: []layout.PaneDef{
					{Title: "vim", Cmd: "vim ."},
					{Title: "shell"},
				},
			},
		},
	}
	out, err := BuildLayout(cfg, "/workdir")
	if err != nil {
		t.Fatalf("BuildLayout: %v", err)
	}
	if !strings.Contains(out, "layout {") {
		t.Fatalf("expected layout root")
	}
	if !strings.Contains(out, "default_tab_template") {
		t.Fatalf("expected default_tab_template")
	}
	if !strings.Contains(out, "tab name=\"editor\"") {
		t.Fatalf("expected editor tab")
	}
	if !strings.Contains(out, "pane name=\"vim\"") {
		t.Fatalf("expected vim pane")
	}
	if !strings.Contains(out, "command=\"$SHELL\"") {
		t.Fatalf("expected command to use $SHELL")
	}
	if !strings.Contains(out, "args \"-lc\" \"vim .\"") {
		t.Fatalf("expected args for vim command")
	}
	if !strings.Contains(out, "cwd=\"/workdir\"") {
		t.Fatalf("expected cwd set to project path")
	}
	if !strings.Contains(out, "pane name=\"shell\"") {
		t.Fatalf("expected shell pane")
	}
}

func TestBuildLayoutFromGrid(t *testing.T) {
	cfg := &layout.LayoutConfig{
		Name:   "grid-layout",
		Grid:   "2x2",
		Window: "grid-tab",
		Titles: []string{"a", "b", "c", "d"},
	}
	out, err := BuildLayout(cfg, "/grid")
	if err != nil {
		t.Fatalf("BuildLayout grid: %v", err)
	}
	if !strings.Contains(out, "tab name=\"grid-tab\"") {
		t.Fatalf("expected grid tab name")
	}
	if !strings.Contains(out, "name=\"a\"") || !strings.Contains(out, "name=\"d\"") {
		t.Fatalf("expected grid pane titles, got:\n%s", out)
	}
}

func TestLayoutAlgorithms(t *testing.T) {
	makeLayout := func(name string) *layout.LayoutConfig {
		return &layout.LayoutConfig{
			Name: name,
			Windows: []layout.WindowDef{
				{
					Name:   "win",
					Layout: name,
					Panes: []layout.PaneDef{
						{Title: "one"},
						{Title: "two"},
						{Title: "three"},
					},
				},
			},
		}
	}

	cases := []struct {
		name      string
		wantSplit string
		wantSize  string
	}{
		{name: "even-horizontal", wantSplit: "Horizontal", wantSize: "34%"},
		{name: "even-vertical", wantSplit: "Vertical", wantSize: "34%"},
		{name: "main-horizontal", wantSplit: "Vertical", wantSize: "60%"},
		{name: "main-vertical", wantSplit: "Horizontal", wantSize: "60%"},
		{name: "tiled", wantSplit: "Vertical", wantSize: ""},
	}

	for _, tc := range cases {
		out, err := BuildLayout(makeLayout(tc.name), "/path")
		if err != nil {
			t.Fatalf("BuildLayout %s: %v", tc.name, err)
		}
		if !strings.Contains(out, "split_direction=\""+tc.wantSplit+"\"") {
			t.Fatalf("%s: expected split_direction %s, got:\n%s", tc.name, tc.wantSplit, out)
		}
		if tc.wantSize != "" && !strings.Contains(out, "size=\""+tc.wantSize+"\"") {
			t.Fatalf("%s: expected size %s, got:\n%s", tc.name, tc.wantSize, out)
		}
	}
}

func TestSplitHelpers(t *testing.T) {
	if parseSplitDirection("vertical") != "Vertical" {
		t.Fatalf("vertical should map to Vertical")
	}
	if parseSplitDirection("h") != "Horizontal" {
		t.Fatalf("h should map to Horizontal")
	}
	if parseSplitDirection("weird") != "Horizontal" {
		t.Fatalf("unknown should default to Horizontal")
	}
	if oppositeSplit("Horizontal") != "Vertical" {
		t.Fatalf("opposite of Horizontal should be Vertical")
	}
	if oppositeSplit("Vertical") != "Horizontal" {
		t.Fatalf("opposite of Vertical should be Horizontal")
	}
}

func TestBuildSequentialTree(t *testing.T) {
	panes := []*paneNode{
		{Name: "one"},
		{Name: "two"},
		{Name: "three"},
	}
	root := buildSequentialTree(panes)
	if root == nil || root.Split == "" {
		t.Fatalf("expected root split for sequential tree")
	}
}
