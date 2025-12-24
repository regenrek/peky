package layout

import "testing"

func TestSanitizeSessionName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"myproject", "myproject"},
		{"MyProject", "myproject"},
		{"my project", "my-project"},
		{"my_project", "my-project"},
		{"my--project", "my-project"},
		{"my@project#123", "myproject123"},
		{"-myproject", "myproject"},
		{"myproject-", "myproject"},
		{"", "session"},
		{"@#$%", "session"},
		{"   ", "session"},
		{" My-Project_123 ", "my-project-123"},
	}
	for _, tc := range cases {
		if got := SanitizeSessionName(tc.input); got != tc.want {
			t.Fatalf("SanitizeSessionName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestResolveSessionName(t *testing.T) {
	cfg := &ProjectLocalConfig{Session: "ConfigSession"}
	if got := ResolveSessionName("/tmp/app", "requested", cfg); got != "requested" {
		t.Fatalf("ResolveSessionName requested = %q", got)
	}
	if got := ResolveSessionName("/tmp/app", "", cfg); got != "ConfigSession" {
		t.Fatalf("ResolveSessionName config = %q", got)
	}
	if got := ResolveSessionName("/tmp/My App", "", nil); got != "my-app" {
		t.Fatalf("ResolveSessionName default = %q", got)
	}
}
