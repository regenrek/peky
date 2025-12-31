package native

import (
	"testing"
	"time"
)

func TestWaitForPaneOutputSignals(t *testing.T) {
	m := NewManager()
	result := make(chan bool, 1)
	go func() {
		result <- m.waitForPaneOutput("p-1", 200*time.Millisecond)
	}()
	time.Sleep(10 * time.Millisecond)
	m.markPaneOutputReady("p-1")

	select {
	case ok := <-result:
		if !ok {
			t.Fatalf("expected wait to return true")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for output signal")
	}
}

func TestWaitForPaneOutputAlreadyReady(t *testing.T) {
	m := NewManager()
	m.markPaneOutputReady("p-2")
	if ok := m.waitForPaneOutput("p-2", 0); !ok {
		t.Fatalf("expected ready pane to return true")
	}
}

func TestWaitForPaneOutputTimeout(t *testing.T) {
	m := NewManager()
	start := time.Now()
	if ok := m.waitForPaneOutput("p-3", 25*time.Millisecond); ok {
		t.Fatalf("expected timeout to return false")
	}
	if time.Since(start) < 20*time.Millisecond {
		t.Fatalf("timeout returned too early")
	}
}
