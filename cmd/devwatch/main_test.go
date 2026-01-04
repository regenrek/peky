package main

import "testing"

func TestParseWatchDirs(t *testing.T) {
	got := parseWatchDirs(" cmd , internal, ,scripts ")
	if len(got) != 3 {
		t.Fatalf("expected 3 dirs, got %d", len(got))
	}
	if got[0] != "cmd" || got[1] != "internal" || got[2] != "scripts" {
		t.Fatalf("unexpected dirs: %#v", got)
	}
}

func TestShouldIgnoreDir(t *testing.T) {
	if !shouldIgnoreDir(".git") {
		t.Fatalf("expected .git to be ignored")
	}
	if shouldIgnoreDir("internal") {
		t.Fatalf("expected internal to be watched")
	}
}

func TestShouldIgnoreFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: "main.go", want: false},
		{path: "notes.txt~", want: true},
		{path: ".#swap", want: true},
		{path: "file.swp", want: true},
		{path: "file.tmp", want: true},
	}
	for _, tc := range cases {
		if got := shouldIgnoreFile(tc.path); got != tc.want {
			t.Fatalf("shouldIgnoreFile(%q)=%v want %v", tc.path, got, tc.want)
		}
	}
}
