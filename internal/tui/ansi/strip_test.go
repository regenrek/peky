package ansi

import "testing"

func TestStrip(t *testing.T) {
	in := "\x1b[31mred\x1b[0m text"
	if got := Strip(in); got != "red text" {
		t.Fatalf("Strip() = %q", got)
	}
}

func TestIsBlank(t *testing.T) {
	if !IsBlank("\x1b[31m \x1b[0m\t") {
		t.Fatalf("IsBlank() should be true")
	}
	if IsBlank("x") {
		t.Fatalf("IsBlank() should be false")
	}
}

func TestLastNonEmpty(t *testing.T) {
	lines := []string{"", "\x1b[31m \x1b[0m", " ok "}
	if got := LastNonEmpty(lines); got != "ok" {
		t.Fatalf("LastNonEmpty() = %q", got)
	}
}

func TestStripHandlesOscAndDcs(t *testing.T) {
	cases := map[string]string{
		"pre\x1b]0;title\x07post":    "prepost", // OSC terminated by BEL
		"pre\x1b]0;title\x1b\\post":  "prepost", // OSC terminated by ST
		"pre\x1bPq\x1b\\post":        "prepost", // DCS terminated by ST
		"pre\x1b^private\x1b\\post":  "prepost", // PM terminated by ST
		"pre\x1b_ignored\x1b\\post":  "prepost", // APC terminated by ST
		"pre\x1bXpayload\x1b\\post":  "prepost", // SOS terminated by ST
		"pre\x1b[31mred\x1b[0mpost":  "preredpost",
		"pre\x1b[38;5;42mhi\x1b[0m":  "prehi",
		"pre\x1b[1;2;3;4;5mtest\x1b": "pretest",
	}
	for input, want := range cases {
		if got := Strip(input); got != want {
			t.Fatalf("Strip(%q) = %q, want %q", input, got, want)
		}
	}
}
