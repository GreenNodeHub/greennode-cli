// Package spinner is a tiny stderr-backed wait animation for the `grn login`
// flow. It redraws a message + a rotating Braille dot frame on a carriage
// return (no newline scroll) until Stop or ctx cancel. Stdlib only — no new
// dependency — so the single-binary, zero-runtime-dependency ethos holds.
package spinner

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// frames is the Braille six-dot cycle used by the login wait animation
// (docker/npm-style). Each frame is one step of the dot rotation.
var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// frameInterval is the redraw period. A package var (not a const) so tests can
// shorten it; mirrors the openBrowser/noisyStderr seam pattern in internal/login.
var frameInterval = 100 * time.Millisecond

// Spinner redraws "<msg> <frame>" on a carriage return to out until Stop or
// ctx cancel. Writes to stderr by construction (the caller passes os.Stderr),
// so machine-readable stdout (--output json) never sees spinner frames.
type Spinner struct {
	ctx      context.Context
	out      io.Writer
	msg      string
	mu       sync.Mutex
	done     chan struct{} // closed by Stop to signal run to exit
	finished chan struct{} // closed by run when it has fully exited
	started  bool          // true once Start has spawned run; guards a second Start
	stop     bool          // true once Stop has run; guards double Stop / Start-after-Stop
}

// New returns a Spinner bound to out and canceled by ctx. New does not start
// the animation — Start does.
func New(ctx context.Context, out io.Writer) *Spinner {
	return &Spinner{
		ctx:      ctx,
		out:      out,
		done:     make(chan struct{}),
		finished: make(chan struct{}),
	}
}

// Start launches the animation goroutine and returns immediately. The first
// frame is drawn synchronously so the user sees activity before the first
// tick. Idempotent: a second Start on the same Spinner is a no-op — it does
// not spawn a second goroutine or rewrite msg. Start after Stop is also a
// no-op.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop || s.started {
		return
	}
	s.started = true
	s.msg = msg
	fmt.Fprintf(s.out, "%s %s", msg, frames[0])
	go s.run()
}

// run is the animating loop. It exits on ctx.Done() or done (Stop), so a
// timeout/Ctrl-C cannot orphan the goroutine even if Stop is never called. On
// exit it closes finished so Stop can block until the goroutine is fully done
// before writing its trailing newline — this prevents a frame being drawn
// after the newline (a stray line on stderr) and removes the bytes.Buffer race
// in tests where the caller reads out after Stop returns.
func (s *Spinner) run() {
	defer close(s.finished)
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.done:
			return
		case <-ticker.C:
			i = (i + 1) % len(frames)
			fmt.Fprintf(s.out, "\r%s %s", s.msg, frames[i])
		}
	}
}

// Stop ends the animation and writes a trailing newline. It blocks until the
// animating goroutine has exited (via finished) so no frame can be drawn after
// the newline. Idempotent: a second Stop is a no-op and writes no further
// output (exactly one newline total). If Start was never called there is no
// goroutine to wait for, so finished is not blocked on.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if s.stop {
		s.mu.Unlock()
		return
	}
	s.stop = true
	close(s.done)
	started := s.started
	s.mu.Unlock()
	// If run already exited (e.g. ctx cancel observed before Stop), finished is
	// already closed and this returns at once. Only run closes finished, so the
	// receive cannot block forever when started is true.
	if started {
		<-s.finished
	}
	fmt.Fprint(s.out, "\n")
}
