package termkeys

import "testing"

func TestIsCopyShortcutKey(t *testing.T) {
	cases := map[string]bool{
		"":                  false,
		"c":                 false,
		"alt+c":             false,
		"ctrl+c":            true,
		"ctrl+shift+c":      true,
		"shift+ctrl+c":      true,
		"cmd+c":             true,
		"command+c":         true,
		"cmd+shift+c":       true,
		"meta+c":            true,
		"super+c":           true,
		"ctrl+insert":       true,
		"ctrl+shift+insert": false,
		"shift+insert":      false,
		"cmd+insert":        false,
	}

	for key, want := range cases {
		if got := IsCopyShortcutKey(key); got != want {
			t.Fatalf("IsCopyShortcutKey(%q) = %v, want %v", key, got, want)
		}
	}
}
