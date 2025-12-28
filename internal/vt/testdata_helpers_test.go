package vt

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func readTestdataSequence(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "vt", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	seq, err := decodeEscapedBytes(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("decode testdata %s: %v", name, err)
	}
	return seq
}

func decodeEscapedBytes(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	decoded, err := strconv.Unquote(`"` + value + `"`)
	if err != nil {
		return nil, err
	}
	return []byte(decoded), nil
}
