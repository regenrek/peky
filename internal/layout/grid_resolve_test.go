package layout

import "testing"

func TestResolveGridCommands(t *testing.T) {
	if got := ResolveGridCommands(nil, 3); len(got) != 0 {
		t.Fatalf("expected empty commands for nil layout")
	}
	cfg := &LayoutConfig{Command: "fallback"}
	got := ResolveGridCommands(cfg, 2)
	if len(got) != 2 || got[0] != "fallback" || got[1] != "fallback" {
		t.Fatalf("ResolveGridCommands fallback = %#v", got)
	}
	cfg = &LayoutConfig{Commands: []string{"a", "b"}, Command: "fallback"}
	got = ResolveGridCommands(cfg, 4)
	want := []string{"a", "b", "fallback", "fallback"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ResolveGridCommands(%d) = %#v", i, got)
		}
	}
}

func TestResolveGridTitles(t *testing.T) {
	if got := ResolveGridTitles(nil, 2); len(got) != 0 {
		t.Fatalf("expected empty titles for nil layout")
	}
	cfg := &LayoutConfig{Titles: []string{"one"}}
	got := ResolveGridTitles(cfg, 3)
	want := []string{"one", "", ""}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ResolveGridTitles(%d) = %#v", i, got)
		}
	}
}
