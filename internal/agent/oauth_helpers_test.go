package agent

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type rcStub struct {
	reader   io.Reader
	readErr  error
	closeErr error
}

func (r *rcStub) Read(p []byte) (int, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	return r.reader.Read(p)
}

func (r *rcStub) Close() error {
	return r.closeErr
}

func TestReadResponseBodyErrors(t *testing.T) {
	if _, err := readResponseBody(nil); err == nil {
		t.Fatalf("expected error for nil response")
	}
	resp := &http.Response{Body: &rcStub{reader: strings.NewReader(""), readErr: errors.New("read")}}
	if _, err := readResponseBody(resp); err == nil {
		t.Fatalf("expected read error")
	}
	resp = &http.Response{Body: &rcStub{reader: strings.NewReader("ok"), closeErr: errors.New("close")}}
	if _, err := readResponseBody(resp); err == nil {
		t.Fatalf("expected close error")
	}
}

func TestReadResponseBodySuccess(t *testing.T) {
	resp := &http.Response{Body: io.NopCloser(strings.NewReader("ok"))}
	out, err := readResponseBody(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Fatalf("out=%q", out)
	}
}

func TestOAuthExpiry(t *testing.T) {
	now := time.Now()
	long := oauthExpiry(3600)
	if long <= now.Add(50*time.Minute).UnixMilli() || long > now.Add(60*time.Minute).UnixMilli() {
		t.Fatalf("long expiry out of range: %d", long)
	}
	short := oauthExpiry(60)
	if short >= now.UnixMilli() {
		t.Fatalf("expected short expiry in past, got %d", short)
	}
}
