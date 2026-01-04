package native

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkOutputLogAppendFlood(b *testing.B) {
	log := newOutputLog(2000)

	var buf bytes.Buffer
	for i := 0; i < 100_000; i++ {
		fmt.Fprintf(&buf, "line%06d\n", i)
	}
	payload := buf.Bytes()

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		log.append(payload)
	}
}
