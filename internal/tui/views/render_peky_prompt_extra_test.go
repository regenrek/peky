package views

import "testing"

func TestViewPekyPromptLine(t *testing.T) {
	m := Model{PekyPromptLine: "hello world"}
	out := m.viewPekyPromptLine(20)
	if out == "" {
		t.Fatalf("expected prompt line output")
	}
	empty := m.viewPekyPromptLine(0)
	if empty != "" {
		t.Fatalf("expected empty for zero width")
	}
}
