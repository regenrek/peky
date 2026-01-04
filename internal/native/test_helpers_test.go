package native

import "testing"

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m
}
