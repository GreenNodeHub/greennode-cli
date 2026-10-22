package testutil

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCaptureStdoutReturnsWrittenText(t *testing.T) {
	out := CaptureStdout(t, func() {
		fmt.Print("hello stdout")
	})
	if out != "hello stdout" {
		t.Errorf("CaptureStdout(...) = %q, want %q", out, "hello stdout")
	}
}

func TestCaptureStderrReturnsWrittenText(t *testing.T) {
	out := CaptureStderr(t, func() {
		fmt.Fprint(os.Stderr, "hello stderr")
	})
	if out != "hello stderr" {
		t.Errorf("CaptureStderr(...) = %q, want %q", out, "hello stderr")
	}
}

func TestCaptureStdoutDoesNotLeakIntoStderr(t *testing.T) {
	out := CaptureStdout(t, func() {
		fmt.Print("stdout-only")
		fmt.Fprint(os.Stderr, "stderr-only")
	})
	if strings.Contains(out, "stderr-only") {
		t.Errorf("CaptureStdout(...) = %q, leaked stderr content", out)
	}
	if !strings.Contains(out, "stdout-only") {
		t.Errorf("CaptureStdout(...) = %q, missing stdout content", out)
	}
}

func TestCaptureStdoutRestoresOnPanic(t *testing.T) {
	before := os.Stdout

	t.Run("panicking capture", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("fn did not panic; test fixture is broken")
			}
		}()
		CaptureStdout(t, func() {
			panic("boom")
		})
	})

	if os.Stdout != before {
		t.Fatalf("os.Stdout not restored after a panicking fn: got %v, want %v", os.Stdout, before)
	}
}

func TestNewTestServerServesHandler(t *testing.T) {
	var gotPath string
	server := NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))

	resp, err := http.Get(server.URL + "/ping")
	if err != nil {
		t.Fatalf("GET %s/ping: %v", server.URL, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if gotPath != "/ping" {
		t.Errorf("handler saw path %q, want /ping", gotPath)
	}
}

func TestNewTestServerClosesOnCleanup(t *testing.T) {
	var url string
	t.Run("server", func(t *testing.T) {
		server := NewTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url = server.URL
	})

	if _, err := http.Get(url); err == nil {
		t.Fatal("server still reachable after its owning subtest completed; NewTestServer did not close it via t.Cleanup")
	}
}
