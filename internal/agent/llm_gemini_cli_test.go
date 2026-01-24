package agent

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseGeminiCLICreds(t *testing.T) {
	if _, err := parseGeminiCLICreds(""); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if _, err := parseGeminiCLICreds("{"); err == nil {
		t.Fatalf("expected error for invalid json")
	}
	if _, err := parseGeminiCLICreds(`{"token":"","projectId":"p"}`); err == nil {
		t.Fatalf("expected error for missing token")
	}
	creds, err := parseGeminiCLICreds(`{"token":"tok","projectId":"proj"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Token != "tok" || creds.ProjectID != "proj" {
		t.Fatalf("creds=%#v", creds)
	}
}

func TestGeminiCLIHeaders(t *testing.T) {
	headers := geminiCLIHeaders("https://sandbox.googleapis.com/v1internal:streamGenerateContent?alt=sse", "tok")
	if headers["Authorization"] != "Bearer tok" {
		t.Fatalf("authorization header missing")
	}
	if headers["User-Agent"] != "antigravity/1.11.5 darwin/arm64" {
		t.Fatalf("sandbox user agent=%q", headers["User-Agent"])
	}
	headers = geminiCLIHeaders("https://generativelanguage.googleapis.com/v1internal:streamGenerateContent?alt=sse", "tok")
	if headers["User-Agent"] != "google-cloud-sdk vscode_cloudshelleditor/0.1" {
		t.Fatalf("non-sandbox user agent=%q", headers["User-Agent"])
	}
	if headers["X-Goog-Api-Client"] != "gl-node/22.17.0" {
		t.Fatalf("non-sandbox api client=%q", headers["X-Goog-Api-Client"])
	}
}

func TestGeminiCLIExtractPayload(t *testing.T) {
	if got := geminiCLIExtractPayload("event: noop"); got != "" {
		t.Fatalf("expected empty payload, got %q", got)
	}
	if got := geminiCLIExtractPayload("data:"); got != "" {
		t.Fatalf("expected empty payload, got %q", got)
	}
	if got := geminiCLIExtractPayload("data: {\"x\":1}"); got != "{\"x\":1}" {
		t.Fatalf("payload=%q", got)
	}
}

func TestGeminiCLIReadStream(t *testing.T) {
	payload1 := `{"response":{"candidates":[{"content":{"parts":[{"text":"Hello "}]},` +
		`"finishReason":"stop"}],"usageMetadata":{"promptTokenCount":5,` +
		`"candidatesTokenCount":3,"thoughtsTokenCount":2,"totalTokenCount":10,` +
		`"cachedContentTokenCount":1}}}`
	payload2 := `{"response":{"candidates":[{"content":{"parts":[{"functionCall":` +
		`{"name":"do","args":{"x":1},"id":"call-1"},"thoughtSignature":"sig"}]},` +
		`"finishReason":"tool_calls"}],"usageMetadata":{"promptTokenCount":0,` +
		`"candidatesTokenCount":0,"thoughtsTokenCount":0,"totalTokenCount":0,` +
		`"cachedContentTokenCount":0}}}`
	stream := strings.NewReader("data: " + payload1 + "\n\n" + "data: " + payload2 + "\n")
	resp, err := geminiCLIReadStream(stream)
	if err != nil {
		t.Fatalf("geminiCLIReadStream error: %v", err)
	}
	if resp.Text != "Hello " {
		t.Fatalf("text=%q", resp.Text)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "do" || resp.ToolCalls[0].ID != "call-1" {
		t.Fatalf("toolcalls=%#v", resp.ToolCalls)
	}
	if resp.Usage.InputTokens != 4 || resp.Usage.OutputTokens != 5 || resp.Usage.TotalTokens != 10 {
		t.Fatalf("usage=%#v", resp.Usage)
	}
}

func TestGeminiCLIParseStreamErrors(t *testing.T) {
	if _, err := geminiCLIParseStream(&http.Response{}); err == nil {
		t.Fatalf("expected error for empty response")
	}
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       ioNopCloser("boom"),
	}
	if _, err := geminiCLIParseStream(resp); err == nil {
		t.Fatalf("expected error for status")
	}
}

func TestGeminiCLIParseStreamEmpty(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser(""),
	}
	out, err := geminiCLIParseStream(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Text != "(no response)" {
		t.Fatalf("expected no response, got %q", out.Text)
	}
}

func ioNopCloser(body string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(body))
}
