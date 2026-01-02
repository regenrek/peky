package terminal

import "testing"

func TestANSICacheSeqDirtyReturnsZero(t *testing.T) {
	w := &Window{}

	w.cacheMu.Lock()
	w.cacheSeq = 42
	w.cacheDirty = true
	w.cacheMu.Unlock()

	if got := w.ANSICacheSeq(); got != 0 {
		t.Fatalf("ANSICacheSeq should return 0 when cache is dirty, got %d", got)
	}

	w.cacheMu.Lock()
	w.cacheDirty = false
	w.cacheMu.Unlock()

	if got := w.ANSICacheSeq(); got != 42 {
		t.Fatalf("ANSICacheSeq should return cached seq when clean, got %d", got)
	}
}
