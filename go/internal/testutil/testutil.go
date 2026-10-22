// Package testutil provides test-only output capture and local HTTP servers.
package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// CaptureStdout captures output and restores stdout even after panic.
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stdout, fn)
}

// CaptureStderr does for os.Stderr what CaptureStdout does for os.Stdout.
func CaptureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stderr, fn)
}

// capture swaps and drains an output pipe.
func capture(t *testing.T, target **os.File, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	original := *target
	*target = writer
	// Register restoration before fn can panic.
	t.Cleanup(func() {
		*target = original
		_ = writer.Close()
		_ = reader.Close()
	})

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	*target = original

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(output)
}

// NewTestServer closes automatically at test cleanup.
func NewTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}
