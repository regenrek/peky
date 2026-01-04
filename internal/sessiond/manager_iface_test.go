package sessiond

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/native"
)

func TestWrapManagerAndWindowNil(t *testing.T) {
	if wrapManager(nil) != nil {
		t.Fatalf("expected nil wrapManager")
	}

	adapter := nativeManagerAdapter{}
	if adapter.Window("pane") != nil {
		t.Fatalf("expected nil window for nil manager")
	}

	mgr, err := native.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer mgr.Close()
	adapter = nativeManagerAdapter{Manager: mgr}
	if adapter.Window("missing") != nil {
		t.Fatalf("expected nil window for missing pane")
	}
}
