package views

import "testing"

func TestChordModsHelpers(t *testing.T) {
	mods := chordMods("ctrl+shift+x")
	if len(mods) != 2 || mods[0] != "ctrl" || mods[1] != "shift" {
		t.Fatalf("unexpected chord mods: %#v", mods)
	}
	if out := stripChordMods("ctrl+shift+x", []string{"ctrl"}); out != "shift+x" {
		t.Fatalf("unexpected stripped chord: %q", out)
	}
	if out := stripChordMods("x", []string{"ctrl"}); out != "x" {
		t.Fatalf("unexpected strip for bare chord: %q", out)
	}
	common := commonChordMods([]string{"ctrl+x", "ctrl+y"})
	if len(common) != 1 || common[0] != "ctrl" {
		t.Fatalf("unexpected common mods: %#v", common)
	}
}
