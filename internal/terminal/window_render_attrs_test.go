package terminal

import "testing"

func TestMaxInt(t *testing.T) {
	if maxInt(1, 2) != 2 || maxInt(3, -1) != 3 {
		t.Fatalf("unexpected maxInt result")
	}
}
