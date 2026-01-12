package dashboard

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/muesli/cancelreader"
)

const tuiTraceInputEnv = "PEKY_TUI_TRACE_INPUT"
const tuiTraceInputFileEnv = "PEKY_TUI_TRACE_INPUT_FILE"
const tuiTraceInputRepairedEnv = "PEKY_TUI_TRACE_INPUT_REPAIRED"
const tuiTraceInputRepairedFileEnv = "PEKY_TUI_TRACE_INPUT_REPAIRED_FILE"

func shouldTraceTUIInput() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(tuiTraceInputEnv)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func shouldTraceTUIInputRepaired() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(tuiTraceInputRepairedEnv)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

type tracedTUIInput struct {
	f     cancelreader.File
	state *tuiInputTraceState
}

func newTracedTUIInput(f cancelreader.File) (*tracedTUIInput, func(), error) {
	return newTracedTUIInputWith(f, tuiInputTracePath(), "tui_input_trace_start")
}

func newTracedTUIInputRepaired(f cancelreader.File) (*tracedTUIInput, func(), error) {
	return newTracedTUIInputWith(f, tuiInputTracePathRepaired(), "tui_input_trace_repaired_start")
}

func newTracedTUIInputWith(f cancelreader.File, tracePath string, startEvent string) (*tracedTUIInput, func(), error) {
	if f == nil {
		return nil, func() {}, fmt.Errorf("nil input file")
	}

	tf, err := os.OpenFile(tracePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open tui input trace file %q: %w", tracePath, err)
	}
	closeFn := func() { _ = tf.Close() }

	state := &tuiInputTraceState{
		event:   startEvent,
		maxTail: 256,
		out:     bufio.NewWriterSize(tf, 64*1024),
	}
	state.noteStart(f)

	return &tracedTUIInput{f: f, state: state}, closeFn, nil
}

func (t *tracedTUIInput) Read(p []byte) (int, error) {
	if t == nil || t.f == nil {
		return 0, io.EOF
	}
	n, err := t.f.Read(p)
	if n > 0 {
		t.state.noteRead(p[:n], len(p), err)
		return n, err
	}
	if err != nil {
		t.state.noteRead(nil, len(p), err)
	}
	return n, err
}

func (t *tracedTUIInput) Write(p []byte) (int, error) {
	if t == nil || t.f == nil {
		return 0, io.ErrClosedPipe
	}
	return t.f.Write(p)
}

func (t *tracedTUIInput) Close() error {
	if t == nil || t.f == nil {
		return nil
	}
	return t.f.Close() // nolint: wrapcheck
}

func (t *tracedTUIInput) Fd() uintptr {
	if t == nil || t.f == nil {
		return 0
	}
	return t.f.Fd()
}

func (t *tracedTUIInput) Name() string {
	if t == nil || t.f == nil {
		return ""
	}
	return t.f.Name()
}

type tuiInputTraceState struct {
	event   string
	maxTail int
	seq     uint64
	tail    []byte

	mu  sync.Mutex
	out *bufio.Writer
}

func (s *tuiInputTraceState) noteStart(f cancelreader.File) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.out == nil {
		return
	}

	now := time.Now()
	line := "ts=" + now.Format(time.RFC3339Nano) +
		" event=" + strconv.Quote(s.event) +
		" pid=" + strconv.Itoa(os.Getpid()) +
		" input_name=" + strconv.Quote(f.Name()) +
		" input_fd=" + strconv.FormatUint(uint64(f.Fd()), 10) +
		"\n"
	_, _ = s.out.WriteString(line)
	_ = s.out.Flush()
}

func (s *tuiInputTraceState) noteRead(buf []byte, readBufLen int, err error) {
	if s == nil {
		return
	}
	s.seq++

	if len(buf) > 0 {
		s.appendTail(buf)
	}
	if len(buf) == 0 && err == nil {
		return
	}
	if len(buf) > 0 && !shouldLogTUIInputChunk(buf) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.out == nil {
		return
	}
	now := time.Now()
	line := "ts=" + now.Format(time.RFC3339Nano) +
		" seq=" + strconv.FormatUint(s.seq, 10) +
		" p_len=" + strconv.Itoa(readBufLen) +
		" n=" + strconv.Itoa(len(buf)) +
		" err=" + strconv.Quote(errString(err)) +
		" chunk_hex=" + strconv.Quote(hexStringPrefix(buf, 96)) +
		" chunk_ascii=" + strconv.Quote(asciiStringPrefix(buf, 96)) +
		" chunk_tail_hex=" + strconv.Quote(hexStringSuffix(buf, 96)) +
		" chunk_tail_ascii=" + strconv.Quote(asciiStringSuffix(buf, 96)) +
		" stream_tail_hex=" + strconv.Quote(hexStringSuffix(s.tail, 96)) +
		" stream_tail_ascii=" + strconv.Quote(asciiStringSuffix(s.tail, 96)) +
		"\n"
	_, _ = s.out.WriteString(line)
	_ = s.out.Flush()
}

func (s *tuiInputTraceState) appendTail(buf []byte) {
	if s == nil || len(buf) == 0 {
		return
	}
	if s.maxTail <= 0 {
		s.maxTail = 256
	}
	if len(buf) >= s.maxTail {
		s.tail = append([]byte(nil), buf[len(buf)-s.maxTail:]...)
		return
	}
	need := len(s.tail) + len(buf)
	if need <= s.maxTail {
		s.tail = append(s.tail, buf...)
		return
	}
	over := need - s.maxTail
	if over >= len(s.tail) {
		s.tail = append([]byte(nil), buf...)
		return
	}
	s.tail = append(append([]byte(nil), s.tail[over:]...), buf...)
}

func shouldLogTUIInputChunk(buf []byte) bool {
	for _, b := range buf {
		switch b {
		case 0x1b: // ESC
			return true
		case '[', '<', ';', 'M', 'm':
			return true
		}
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			return true
		}
	}
	return false
}

func tuiInputTracePath() string {
	if p := strings.TrimSpace(os.Getenv(tuiTraceInputFileEnv)); p != "" {
		return p
	}
	if dir := strings.TrimSpace(os.Getenv("PEKY_DATA_DIR")); dir != "" {
		return filepath.Join(dir, "tui-input-trace.log")
	}
	return fmt.Sprintf("/tmp/peky-tui-input-trace-%d.log", os.Getpid())
}

func tuiInputTracePathRepaired() string {
	if p := strings.TrimSpace(os.Getenv(tuiTraceInputRepairedFileEnv)); p != "" {
		return p
	}
	if dir := strings.TrimSpace(os.Getenv("PEKY_DATA_DIR")); dir != "" {
		return filepath.Join(dir, "tui-input-trace-repaired.log")
	}
	return fmt.Sprintf("/tmp/peky-tui-input-trace-repaired-%d.log", os.Getpid())
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func slicePrefix(buf []byte, max int) ([]byte, bool) {
	if max <= 0 || len(buf) <= max {
		return buf, false
	}
	return buf[:max], true
}

func sliceSuffix(buf []byte, max int) ([]byte, bool) {
	if max <= 0 || len(buf) <= max {
		return buf, false
	}
	return buf[len(buf)-max:], true
}

func hexStringPrefix(buf []byte, max int) string {
	if len(buf) == 0 {
		return ""
	}
	slice, truncated := slicePrefix(buf, max)
	dst := make([]byte, hex.EncodedLen(len(slice)))
	hex.Encode(dst, slice)
	var out bytes.Buffer
	for i := 0; i < len(dst); i += 2 {
		if i > 0 {
			_ = out.WriteByte(' ')
		}
		_, _ = out.Write(dst[i : i+2])
	}
	if truncated {
		_, _ = out.WriteString(" …")
	}
	return out.String()
}

func hexStringSuffix(buf []byte, max int) string {
	if len(buf) == 0 {
		return ""
	}
	slice, truncated := sliceSuffix(buf, max)
	dst := make([]byte, hex.EncodedLen(len(slice)))
	hex.Encode(dst, slice)
	var out bytes.Buffer
	if truncated {
		_, _ = out.WriteString("… ")
	}
	for i := 0; i < len(dst); i += 2 {
		if i > 0 {
			_ = out.WriteByte(' ')
		}
		_, _ = out.Write(dst[i : i+2])
	}
	return out.String()
}

func asciiStringPrefix(buf []byte, max int) string {
	if len(buf) == 0 {
		return ""
	}
	slice, truncated := slicePrefix(buf, max)
	var out bytes.Buffer
	out.Grow(len(slice))
	for _, b := range slice {
		if b >= 0x20 && b < 0x7f {
			_ = out.WriteByte(b)
			continue
		}
		switch b {
		case '\n':
			_, _ = out.WriteString(`\n`)
		case '\r':
			_, _ = out.WriteString(`\r`)
		case '\t':
			_, _ = out.WriteString(`\t`)
		default:
			_, _ = out.WriteString(fmt.Sprintf("\\x%02x", b))
		}
	}
	if truncated {
		_, _ = out.WriteString(" …")
	}
	return out.String()
}

func asciiStringSuffix(buf []byte, max int) string {
	if len(buf) == 0 {
		return ""
	}
	slice, truncated := sliceSuffix(buf, max)
	var out bytes.Buffer
	out.Grow(len(slice))
	if truncated {
		_, _ = out.WriteString("… ")
	}
	for _, b := range slice {
		if b >= 0x20 && b < 0x7f {
			_ = out.WriteByte(b)
			continue
		}
		switch b {
		case '\n':
			_, _ = out.WriteString(`\n`)
		case '\r':
			_, _ = out.WriteString(`\r`)
		case '\t':
			_, _ = out.WriteString(`\t`)
		default:
			_, _ = out.WriteString(fmt.Sprintf("\\x%02x", b))
		}
	}
	return out.String()
}
