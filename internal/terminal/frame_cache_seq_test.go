package terminal

import "testing"

func TestFrameCacheSeqDirtyReturnsZero(t *testing.T) {
	w := &Window{}

	w.cacheMu.Lock()
	w.cacheSeq = 42
	w.cacheDirty = true
	w.cacheMu.Unlock()

	if got := w.FrameCacheSeq(); got != 0 {
		t.Fatalf("FrameCacheSeq should return 0 when cache is dirty, got %d", got)
	}

	w.cacheMu.Lock()
	w.cacheDirty = false
	w.cacheMu.Unlock()

	if got := w.FrameCacheSeq(); got != 42 {
		t.Fatalf("FrameCacheSeq should return cached seq when clean, got %d", got)
	}
}
