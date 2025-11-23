package layout

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		rows    int
		cols    int
		wantErr bool
	}{
		{name: "default", spec: "", rows: Default.Rows, cols: Default.Columns},
		{name: "spaces", spec: " 2x3 ", rows: 2, cols: 3},
		{name: "big", spec: "3x4", rows: 3, cols: 4},
		{name: "bad format", spec: "abc", wantErr: true},
		{name: "zero", spec: "0x2", wantErr: true},
		{name: "too many", spec: "4x4", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := Parse(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) expected error", tt.spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.spec, err)
			}
			if g.Rows != tt.rows || g.Columns != tt.cols {
				t.Fatalf("Parse(%q) got %dx%d", tt.spec, g.Rows, g.Columns)
			}
		})
	}
}
