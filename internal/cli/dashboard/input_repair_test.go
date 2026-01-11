package dashboard

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/muesli/cancelreader"
)

type fakeCancelFile struct {
	name  string
	fd    uintptr
	reads [][]byte
	i     int
}

var _ cancelreader.File = (*fakeCancelFile)(nil)

func (f *fakeCancelFile) Read(p []byte) (int, error) {
	if f.i >= len(f.reads) {
		return 0, io.EOF
	}
	chunk := f.reads[f.i]
	f.i++
	n := copy(p, chunk)
	return n, nil
}

func (f *fakeCancelFile) Write(p []byte) (int, error) { return len(p), nil }

func (f *fakeCancelFile) Close() error { return nil }

func (f *fakeCancelFile) Fd() uintptr { return f.fd }

func (f *fakeCancelFile) Name() string { return f.name }

func TestSafeReadLenCutsBeforeIncompleteEscape(t *testing.T) {
	out := append(bytes.Repeat([]byte("A"), 252), escByte, '[', '<', '6')
	if got := safeReadLen(out, 256); got != 252 {
		t.Fatalf("safeReadLen=%d", got)
	}
}

func TestSafeReadLenReturns1ForSingleEsc(t *testing.T) {
	if got := safeReadLen([]byte{escByte}, 256); got != 1 {
		t.Fatalf("safeReadLen=%d", got)
	}
}

func TestRepairedTUIInputCoalescesSplitSGRMouse(t *testing.T) {
	prefix := bytes.Repeat([]byte("A"), 252)
	part1 := append(append([]byte(nil), prefix...), escByte, '[', '<', '6')
	part2 := []byte("4;68;26M")

	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{part1, part2}}
	r := newRepairedTUIInput(f)
	r.readyFn = func(uintptr, time.Duration) (bool, error) { return true, nil }

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read 1 err=%v", err)
	}
	if n != len(prefix) {
		t.Fatalf("Read 1 n=%d", n)
	}
	if !bytes.Equal(buf[:n], prefix) {
		t.Fatalf("Read 1 mismatch")
	}

	n, err = r.Read(buf)
	if err != nil {
		t.Fatalf("Read 2 err=%v", err)
	}
	want := []byte("\x1b[<64;68;26M")
	if n != len(want) {
		t.Fatalf("Read 2 n=%d", n)
	}
	if !bytes.Equal(buf[:n], want) {
		t.Fatalf("Read 2 mismatch: %q", buf[:n])
	}

	n, err = r.Read(buf)
	if n != 0 || err == nil || err != io.EOF {
		t.Fatalf("Read 3 n=%d err=%v", n, err)
	}
}

func TestRepairedTUIInputFlushesEscWhenNoMoreBytes(t *testing.T) {
	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{{escByte}}}
	r := newRepairedTUIInput(f)
	r.readyFn = func(uintptr, time.Duration) (bool, error) { return false, nil }

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read 1 err=%v", err)
	}
	if n != 1 || buf[0] != escByte {
		t.Fatalf("Read 1 n=%d buf=% x", n, buf[:n])
	}

	n, err = r.Read(buf)
	if n != 0 || err == nil || err != io.EOF {
		t.Fatalf("Read 2 n=%d err=%v", n, err)
	}
}

func TestRepairedTUIInputDropsIncompleteSGRMouseOnTimeout(t *testing.T) {
	prefix := bytes.Repeat([]byte("A"), 252)
	part1 := append(append([]byte(nil), prefix...), escByte, '[', '<', '6')

	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{part1}}
	r := newRepairedTUIInput(f)
	r.readyFn = func(uintptr, time.Duration) (bool, error) { return false, nil }

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read 1 err=%v", err)
	}
	if n != len(prefix) {
		t.Fatalf("Read 1 n=%d", n)
	}
	if !bytes.Equal(buf[:n], prefix) {
		t.Fatalf("Read 1 mismatch")
	}

	n, err = r.Read(buf)
	if n != 0 || err == nil || err != io.EOF {
		t.Fatalf("Read 2 n=%d err=%v", n, err)
	}
}

func TestRepairedTUIInputDropsDanglingEscBracketOnTimeout(t *testing.T) {
	prefix := bytes.Repeat([]byte("A"), 252)
	part1 := append(append([]byte(nil), prefix...), escByte, '[')

	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{part1}}
	r := newRepairedTUIInput(f)
	r.readyFn = func(uintptr, time.Duration) (bool, error) { return false, nil }

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read 1 err=%v", err)
	}
	if n != len(prefix) {
		t.Fatalf("Read 1 n=%d", n)
	}
	if !bytes.Equal(buf[:n], prefix) {
		t.Fatalf("Read 1 mismatch")
	}

	n, err = r.Read(buf)
	if n != 0 || err == nil || err != io.EOF {
		t.Fatalf("Read 2 n=%d err=%v", n, err)
	}
}

func TestRepairedTUIInputRepairsMouseAfterDroppedEscBracketPrefix(t *testing.T) {
	f := &fakeCancelFile{
		fd:   7,
		name: "fake",
		reads: [][]byte{
			{escByte, '['},
			[]byte("<64;68;26M"),
		},
	}
	r := newRepairedTUIInput(f)
	r.readyFn = func(uintptr, time.Duration) (bool, error) { return false, nil }

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read 1 err=%v", err)
	}
	want := []byte("\x1b[<64;68;26M")
	if n != len(want) {
		t.Fatalf("Read 1 n=%d", n)
	}
	if !bytes.Equal(buf[:n], want) {
		t.Fatalf("Read 1 mismatch: %q", buf[:n])
	}

	n, err = r.Read(buf)
	if n != 0 || err == nil || err != io.EOF {
		t.Fatalf("Read 2 n=%d err=%v", n, err)
	}
}

func TestRepairedTUIInputRepairsMissingEscForSGRMouse(t *testing.T) {
	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{[]byte("<65;70;16M")}}
	r := newRepairedTUIInput(f)

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read err=%v", err)
	}
	want := []byte("\x1b[<65;70;16M")
	if n != len(want) {
		t.Fatalf("Read n=%d", n)
	}
	if !bytes.Equal(buf[:n], want) {
		t.Fatalf("Read mismatch: %q", buf[:n])
	}
}

func TestRepairedTUIInputRepairsBracketBurstMissingEscForSGRMouse(t *testing.T) {
	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{[]byte("[[<65;70;16M")}}
	r := newRepairedTUIInput(f)

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read err=%v", err)
	}
	want := []byte("\x1b[<65;70;16M")
	if n != len(want) {
		t.Fatalf("Read n=%d", n)
	}
	if !bytes.Equal(buf[:n], want) {
		t.Fatalf("Read mismatch: %q", buf[:n])
	}
}

func TestRepairedTUIInputDropsSGRMouseFragment(t *testing.T) {
	in := []byte("5;71;16M\x1b[<65;71;16M")
	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{in}}
	r := newRepairedTUIInput(f)

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read err=%v", err)
	}
	want := []byte("\x1b[<65;71;16M")
	if n != len(want) {
		t.Fatalf("Read n=%d", n)
	}
	if !bytes.Equal(buf[:n], want) {
		t.Fatalf("Read mismatch: %q", buf[:n])
	}
}

func TestRepairedTUIInputDropsSGRMouseTailFragments(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "cx_cy",
			in:   []byte("66;21M\x1b[<64;66;21M"),
			want: []byte("\x1b[<64;66;21M"),
		},
		{
			name: "missing_cb_and_cx",
			in:   []byte(";21M\x1b[<65;66;21M"),
			want: []byte("\x1b[<65;66;21M"),
		},
		{
			name: "missing_cb_cx_cy",
			in:   []byte("1M\x1b[<65;66;21M"),
			want: []byte("\x1b[<65;66;21M"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{tt.in}}
			r := newRepairedTUIInput(f)

			buf := make([]byte, 256)
			n, err := r.Read(buf)
			if err != nil {
				t.Fatalf("Read err=%v", err)
			}
			if n != len(tt.want) {
				t.Fatalf("Read n=%d", n)
			}
			if !bytes.Equal(buf[:n], tt.want) {
				t.Fatalf("Read mismatch: %q", buf[:n])
			}
		})
	}
}

func TestRepairedTUIInputNeverReturnsZeroNilForIncompleteCSI(t *testing.T) {
	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{{escByte, '['}}}
	r := newRepairedTUIInput(f)
	r.readyFn = func(uintptr, time.Duration) (bool, error) { return false, nil }

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if n == 0 && err == nil {
		t.Fatalf("Read returned (0, nil)")
	}
}

func TestRepairedTUIInputDoesNotDropBracketedANSIText(t *testing.T) {
	in := []byte("[1;31mhello")
	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{in}}
	r := newRepairedTUIInput(f)

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read err=%v", err)
	}
	if !bytes.Equal(buf[:n], in) {
		t.Fatalf("Read mismatch: %q", buf[:n])
	}
}

func TestRepairedTUIInputDropsDanglingBracketDuringMouseBurst(t *testing.T) {
	f := &fakeCancelFile{fd: 7, name: "fake", reads: [][]byte{{'['}}}
	r := newRepairedTUIInput(f)
	r.lastMouseSeqAt = time.Now()
	r.readyFn = func(uintptr, time.Duration) (bool, error) { return false, nil }

	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if n != 0 || err == nil || err != io.EOF {
		t.Fatalf("Read n=%d err=%v", n, err)
	}
}

func TestScanEscapeSequenceHandlesESCBracketBracket(t *testing.T) {
	b := []byte("\x1b[[1~")
	end, ok := scanEscapeSequence(b, 0)
	if !ok {
		t.Fatalf("scanEscapeSequence ok=false")
	}
	if end != len(b) {
		t.Fatalf("scanEscapeSequence end=%d", end)
	}
}
