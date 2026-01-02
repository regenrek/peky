package limits

import "testing"

func TestNormalize(t *testing.T) {
	cols, rows := Normalize(0, -2)
	if cols != 1 || rows != 1 {
		t.Fatalf("Normalize = %dx%d, want 1x1", cols, rows)
	}
}

func TestClamp(t *testing.T) {
	cols, rows := Clamp(PaneMaxCols+10, PaneMaxRows+10)
	if cols != PaneMaxCols || rows != PaneMaxRows {
		t.Fatalf("Clamp = %dx%d, want %dx%d", cols, rows, PaneMaxCols, PaneMaxRows)
	}
}

func TestValidateMax(t *testing.T) {
	if err := ValidateMax(PaneMaxCols, PaneMaxRows); err != nil {
		t.Fatalf("ValidateMax unexpected error: %v", err)
	}
	if err := ValidateMax(PaneMaxCols+1, PaneMaxRows); err == nil {
		t.Fatalf("ValidateMax expected error for cols")
	}
}
