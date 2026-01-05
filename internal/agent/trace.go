package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/regenrek/peakypanes/internal/appdirs"
)

const traceFieldLimit = 8000

type traceLogger struct {
	mu   sync.Mutex
	enc  *json.Encoder
	file *lumberjack.Logger
}

type traceEvent struct {
	Time       string         `json:"time"`
	RunID      string         `json:"run_id"`
	Event      string         `json:"event"`
	Step       int            `json:"step,omitempty"`
	Provider   string         `json:"provider,omitempty"`
	Model      string         `json:"model,omitempty"`
	Prompt     string         `json:"prompt,omitempty"`
	Context    string         `json:"context,omitempty"`
	Text       string         `json:"text,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Usage      Usage          `json:"usage,omitempty"`
	ToolCall   *ToolCall      `json:"tool_call,omitempty"`
	ToolResult *ToolResult    `json:"tool_result,omitempty"`
	Error      string         `json:"error,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
}

func DefaultTracePath() (string, error) {
	dir, err := appdirs.RuntimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "peky-agent.log"), nil
}

func newTraceLogger(path string) (*traceLogger, func() error, error) {
	if path == "" {
		return nil, func() error { return nil }, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, fmt.Errorf("trace log dir: %w", err)
	}
	rot := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     7,
		Compress:   true,
	}
	return &traceLogger{enc: json.NewEncoder(rot), file: rot}, rot.Close, nil
}

func (t *traceLogger) log(event traceEvent) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_ = t.enc.Encode(event)
}

func truncateField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= traceFieldLimit {
		return value
	}
	return value[:traceFieldLimit] + "…"
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
