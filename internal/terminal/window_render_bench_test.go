package terminal

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/vt"
)

func BenchmarkWindowRender(b *testing.B) {
	sizes := []struct {
		name string
		cols int
		rows int
	}{
		{name: "80x24", cols: 80, rows: 24},
		{name: "120x40", cols: 120, rows: 40},
	}

	for _, size := range sizes {
		payload := benchPayload(size.cols, size.rows)

		b.Run("ANSIRefresh/"+size.name, func(b *testing.B) {
			w := newBenchWindow(size.cols, size.rows)
			seedBenchWindow(w, payload)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w.refreshANSICache()
			}
		})

		b.Run("Lipgloss/"+size.name, func(b *testing.B) {
			w := newBenchWindow(size.cols, size.rows)
			seedBenchWindow(w, payload)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = w.ViewLipglossCtx(context.Background(), false, termenv.TrueColor)
			}
		})

		b.Run("LipglossCursor/"+size.name, func(b *testing.B) {
			w := newBenchWindow(size.cols, size.rows)
			seedBenchWindow(w, payload)
			w.cursorVisible.Store(true)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = w.ViewLipglossCtx(context.Background(), true, termenv.TrueColor)
			}
		})
	}
}

func newBenchWindow(cols, rows int) *Window {
	term := vt.NewEmulator(cols, rows)
	w := &Window{
		term:       term,
		cols:       cols,
		rows:       rows,
		updates:    make(chan struct{}, 1),
		renderCh:   make(chan struct{}, 1),
		cacheDirty: true,
	}
	w.cursorVisible.Store(true)
	w.lastUpdate.Store(time.Now().UnixNano())
	return w
}

func seedBenchWindow(w *Window, payload []byte) {
	if w == nil || w.term == nil {
		return
	}
	w.termMu.Lock()
	_, _ = w.term.Write(payload)
	w.termMu.Unlock()
	w.markDirty()
}

func benchPayload(cols, rows int) []byte {
	if cols <= 0 {
		cols = 1
	}
	if rows <= 0 {
		rows = 1
	}
	var b strings.Builder
	b.Grow(rows * (cols + 16))
	for y := 0; y < rows; y++ {
		r := (y * 37) % 255
		g := (y * 97) % 255
		bl := (y * 13) % 255
		b.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, bl))
		for x := 0; x < cols; x++ {
			b.WriteByte(byte('a' + (x+y)%26))
		}
		b.WriteString("\x1b[0m")
		if y < rows-1 {
			b.WriteByte('\n')
		}
	}
	return []byte(b.String())
}
