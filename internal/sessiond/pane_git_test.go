package sessiond

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestPaneGitCacheMetaIsNonBlocking(t *testing.T) {
	cache := newPaneGitCache()

	probeStarted := make(chan struct{}, 1)
	unblock := make(chan struct{})
	cache.probe = func(ctx context.Context, cwd string) (PaneGitMeta, bool) {
		select {
		case probeStarted <- struct{}{}:
		default:
		}
		<-unblock
		return PaneGitMeta{Root: "/repo", Branch: "main"}, true
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var wg sync.WaitGroup
	cache.Start(ctx, &wg, 1)

	var (
		meta PaneGitMeta
		ok   bool
	)
	metaDone := make(chan struct{})
	go func() {
		meta, ok = cache.Meta(context.Background(), "/tmp/repo")
		close(metaDone)
	}()
	select {
	case <-metaDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Meta blocked")
	}
	if ok || meta.Root != "" {
		t.Fatalf("expected empty cache value, got ok=%v meta=%#v", ok, meta)
	}

	select {
	case <-probeStarted:
	case <-time.After(time.Second):
		t.Fatal("expected probe to start")
	}

	close(unblock)

	deadline := time.Now().Add(2 * time.Second)
	for {
		meta, ok = cache.Meta(context.Background(), "/tmp/repo")
		if ok && meta.Root == "/repo" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected cached meta, got ok=%v meta=%#v", ok, meta)
		}
		runtime.Gosched()
	}

	cancel()
	wg.Wait()
}
