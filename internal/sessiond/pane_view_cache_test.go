package sessiond

import (
	"testing"
	"time"
)

func TestPaneViewCacheEvictsToCap(t *testing.T) {
	c := &clientConn{
		paneViewCache: make(map[paneViewCacheKey]cachedPaneView),
	}
	for i := 0; i < paneViewCacheMaxEntries+5; i++ {
		key := paneViewCacheKey{
			PaneID: "p-1",
			Cols:   i + 1,
			Rows:   1,
		}
		c.paneViewCachePut(key, PaneViewResponse{UpdateSeq: uint64(i + 1)})
	}

	c.paneViewCacheMu.Lock()
	n := len(c.paneViewCache)
	c.paneViewCacheMu.Unlock()
	if n > paneViewCacheMaxEntries {
		t.Fatalf("expected cache size <= %d, got %d", paneViewCacheMaxEntries, n)
	}
}

func TestPaneViewCacheTTLExpires(t *testing.T) {
	c := &clientConn{
		paneViewCache: make(map[paneViewCacheKey]cachedPaneView),
	}
	key := paneViewCacheKey{
		PaneID: "p-1",
		Cols:   80,
		Rows:   24,
	}
	c.paneViewCachePut(key, PaneViewResponse{UpdateSeq: 1})

	c.paneViewCacheMu.Lock()
	entry := c.paneViewCache[key]
	entry.renderedAt = time.Now().Add(-paneViewCacheTTL * 2)
	c.paneViewCache[key] = entry
	c.paneViewCacheMu.Unlock()

	if _, ok := c.paneViewCacheGet(key); ok {
		t.Fatalf("expected expired entry to be purged")
	}
}

func TestPaneViewCacheGetRefreshesAccessTime(t *testing.T) {
	c := &clientConn{
		paneViewCache: make(map[paneViewCacheKey]cachedPaneView),
	}
	key := paneViewCacheKey{
		PaneID: "p-1",
		Cols:   80,
		Rows:   24,
	}
	c.paneViewCachePut(key, PaneViewResponse{UpdateSeq: 1})

	c.paneViewCacheMu.Lock()
	entry := c.paneViewCache[key]
	entry.renderedAt = time.Now().Add(-paneViewCacheTTL / 2)
	c.paneViewCache[key] = entry
	c.paneViewCacheMu.Unlock()

	if _, ok := c.paneViewCacheGetEntry(key); !ok {
		t.Fatalf("expected entry to be present")
	}

	c.paneViewCacheMu.Lock()
	updated := c.paneViewCache[key]
	c.paneViewCacheMu.Unlock()
	if !updated.renderedAt.After(entry.renderedAt) {
		t.Fatalf("expected renderedAt to refresh")
	}
}
