package spec

import "testing"

func TestLoadDefaultSpec(t *testing.T) {
	spec, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() err=%v", err)
	}
	if spec.App.Name == "" {
		t.Fatalf("expected app name set")
	}
	if len(spec.Commands) == 0 {
		t.Fatalf("expected commands")
	}
}

func TestValidateRejectsEmpty(t *testing.T) {
	if err := Validate([]byte("")); err == nil {
		t.Fatalf("expected error for empty spec")
	}
}
