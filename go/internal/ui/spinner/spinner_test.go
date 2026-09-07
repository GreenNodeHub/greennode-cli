package spinner

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

// setFastInterval speeds up the test; restored on cleanup.
func setFastInterval(t *testing.T, d time.Duration) {
	t.Helper()
	old := frameInterval
	frameInterval = d
	t.Cleanup(func() { frameInterval = old })
}

func TestSpinner_CyclesFramesAndStopsWithNewline(t *testing.T) {
	setFastInterval(t, 1*time.Millisecond)
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := New(ctx, &buf)
	s.Start("waiting")
	// Let several frames tick.
	time.Sleep(20 * time.Millisecond)
	s.Stop()

	out := buf.String()
	if !strings.Contains(out, "waiting") {
		t.Errorf("output missing message: %q", out)
	}
	// First frame is drawn immediately.
	if i := strings.Index(out, "⠋"); i < 0 {
		t.Errorf("output missing first frame ⠋: %q", out)
	} else if j := strings.Index(out, "⠙"); j >= 0 && j < i {
		t.Errorf("⠙ (%d) appeared before first ⠋ (%d): %q", j, i, out)
	}
	// Stop writes exactly one trailing newline.
	if strings.Count(out, "\n") != 1 {
		t.Errorf("want exactly one newline, got %d in %q", strings.Count(out, "\n"), out)
	}
}

func TestSpinner_StopIsIdempotent(t *testing.T) {
	setFastInterval(t, 1*time.Millisecond)
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := New(ctx, &buf)
	s.Start("x")
	s.Stop()
	s.Stop() // must not panic and must not write a second newline

	if strings.Count(buf.String(), "\n") != 1 {
		t.Errorf("double Stop wrote %d newlines, want 1: %q", strings.Count(buf.String(), "\n"), buf.String())
	}
}

// TestSpinner_CtxCancelStopsLoop proves the goroutine exits within ~1 tick of
// ctx cancel even if Stop is never called (a timeout/Ctrl-C during the wait).
func TestSpinner_CtxCancelStopsLoop(t *testing.T) {
	setFastInterval(t, 1*time.Millisecond)
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	s := New(ctx, &buf)
	s.Start("x")
	time.Sleep(5 * time.Millisecond) // let it tick
	cancel()
	<-s.finished                     // wait for run to fully exit on ctx cancel (deterministic sync; a sleep has no happens-before edge to run's writes)
	n := buf.Len()
	time.Sleep(30 * time.Millisecond) // would tick ~30× more if the loop were still alive
	if buf.Len() != n {
		t.Errorf("buffer grew after ctx cancel: was %d, now %d (loop did not exit)", n, buf.Len())
	}
}
