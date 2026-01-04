package logging

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/regenrek/peakypanes/internal/limits"
)

var includePayloads atomic.Bool

func setIncludePayloads(v bool) {
	includePayloads.Store(v)
}

func IncludePayloads() bool {
	return includePayloads.Load()
}

// PayloadAttr returns a safe payload attribute for logging.
// By default it redacts payload bytes and includes only length + hash.
func PayloadAttr(key string, payload []byte) slog.Attr {
	if key == "" {
		key = "payload"
	}
	if len(payload) == 0 {
		return slog.String(key, `""`)
	}
	if !IncludePayloads() {
		return slog.String(key, redactedPayloadString(payload))
	}
	const preview = 256
	if len(payload) <= preview {
		return slog.String(key, fmt.Sprintf("%q", payload))
	}
	head := payload[:preview]
	return slog.String(key, fmt.Sprintf("%q...(+%d bytes)", head, len(payload)-preview))
}

func redactedPayloadString(payload []byte) string {
	data := payload
	limit := limits.PayloadInspectLimit
	prefixLen := len(payload)
	if limit > 0 && len(data) > limit {
		data = data[:limit]
		prefixLen = limit
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if len(hash) > 12 {
		hash = hash[:12]
	}
	return fmt.Sprintf("redacted(len=%d sha256_prefix=%s prefix_len=%d)", len(payload), hash, prefixLen)
}
