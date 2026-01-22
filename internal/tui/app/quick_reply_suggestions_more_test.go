package app

import "testing"

func TestBroadcastDescFromGroup(t *testing.T) {
	if got := broadcastDescFromGroup(commandGroup{}); got != defaultBroadcastDesc {
		t.Fatalf("empty group = %q", got)
	}
	group := commandGroup{Commands: []commandSpec{{Desc: " send to all "}}}
	if got := broadcastDescFromGroup(group); got != "send to all" {
		t.Fatalf("desc = %q", got)
	}
	group = commandGroup{Commands: []commandSpec{{Label: " broadcast "}}}
	if got := broadcastDescFromGroup(group); got != "broadcast" {
		t.Fatalf("label = %q", got)
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	if got := longestCommonPrefix(nil); got != "" {
		t.Fatalf("nil prefix = %q", got)
	}
	if got := longestCommonPrefix([]string{"alpha", "alpine", "ally"}); got != "al" {
		t.Fatalf("prefix = %q", got)
	}
	if got := longestCommonPrefix([]string{"alpha", "beta"}); got != "" {
		t.Fatalf("expected empty prefix, got %q", got)
	}
}

func TestExactMatch(t *testing.T) {
	if exactMatch("", []string{"a"}) {
		t.Fatalf("expected false for empty prefix")
	}
	if !exactMatch("beta", []string{"alpha", "beta"}) {
		t.Fatalf("expected true for exact match")
	}
	if exactMatch("gamma", []string{"alpha", "beta"}) {
		t.Fatalf("expected false for missing match")
	}
}
