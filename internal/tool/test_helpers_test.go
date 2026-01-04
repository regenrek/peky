package tool

import "testing"

func defaultRegistry(t *testing.T) *Registry {
	t.Helper()
	reg, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	return reg
}
