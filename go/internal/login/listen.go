package login

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// ErrAuthzDenied is returned by Serve when the callback carries RFC 6749
// §4.1.2.1 error params (?error=…): the user denied consent or IAM rejected the
// authorize request.
type ErrAuthzDenied struct {
	Code        string
	Description string
}

func (e *ErrAuthzDenied) Error() string {
	return fmt.Sprintf("login: authorization denied (code=%s): %s", e.Code, e.Description)
}

const callbackPath = "/callback"

// Listener is a one-shot loopback HTTP server that captures the IAM callback.
// It binds 127.0.0.1:<random port>/callback, serves exactly one request, then
// shuts down (RFC 8252 loopback redirect URI).
type Listener struct {
	uri       string
	srv       *http.Server
	bound     net.Listener
	mu        sync.Mutex
	code      string
	state     string
	cbErr     error
	wantState string        // when non-empty, handle rejects callbacks whose appState differs
	received  chan struct{} // closed on first callback
}

// setExpectedState arms the state-mismatch check. When set to a non-empty nonce,
// handle serves a failure page and Serve returns ErrStateMismatch if the
// callback's appState differs from it. The zero value (empty) disables the
// check, leaving the listener nonce-agnostic (how the unit tests exercise it).
func (l *Listener) setExpectedState(s string) { l.wantState = s }

// NewListener binds 127.0.0.1:0 (random ephemeral port) and returns a Listener
// whose RedirectURI() is http://127.0.0.1:<port>/callback. Serve starts
// accepting.
func NewListener() (*Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("login: bind loopback: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	l := &Listener{
		uri:      fmt.Sprintf("http://127.0.0.1:%d/callback", port),
		bound:    ln,
		received: make(chan struct{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, l.handle)
	l.srv = &http.Server{Handler: mux}
	return l, nil
}

// RedirectURI returns the URI to put on the IAM authorize URL.
func (l *Listener) RedirectURI() string { return l.uri }

// Close releases the bound listener. Safe to call multiple times (idempotent).
func (l *Listener) Close() error {
	if l.bound != nil {
		err := l.bound.Close()
		l.bound = nil
		return err
	}
	return nil
}

// Serve runs the server until the first callback arrives or ctx is cancelled.
// Returns the parsed code and appState; RFC 6749 error params → *ErrAuthzDenied.
func (l *Listener) Serve(ctx context.Context) (string, string, error) {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- l.srv.Serve(l.bound) // ErrServerClosed on Shutdown
	}()

	ctxCancel := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = l.srv.Shutdown(context.Background())
			close(ctxCancel)
		case <-l.received:
			// callback arrived; Serve's <-received branch drains srv.Serve.
		}
	}()

	select {
	case <-l.received:
		// handle() captured code/state/cbErr and triggered Shutdown.
		<-serveErr // wait for the server goroutine to finish
		return l.code, l.state, l.cbErr
	case <-ctxCancel:
		<-serveErr
		return "", "", ctx.Err()
	case <-serveErr:
		// Server exited on its own (e.g. accept error) without a callback.
		// On ctx cancel, Shutdown (from the watcher) closes the listener and
		// srv.Serve returns here before ctxCancel is observed — surface the
		// honest ctx error so errors.Is(., context.Canceled) holds.
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		return "", "", fmt.Errorf("login: listener exited without callback: %w", l.cbErr)
	}
}

// handle is the /callback handler. Extracts code+appState (or error params),
// serves a static result page, and signals Serve to return. One-shot: after the
// first request it shuts the server down.
func (l *Listener) handle(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Shutdown in a goroutine: calling (*http.Server).Shutdown synchronously
	// from within the handler self-deadlocks (Shutdown waits for this in-flight
	// connection to drain, which can't happen until handle returns). Running it
	// async lets handle return, the response flush, and the connection go idle so
	// the Shutdown goroutine then completes. Triggered by handle (one-shot).
	defer func() { go l.srv.Shutdown(context.Background()) }()
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		l.cbErr = &ErrAuthzDenied{Code: e, Description: q.Get("error_description")}
		writeResult(w, false, "Login failed. You may close this tab.")
		l.signal()
		return
	}
	l.code = q.Get("code")
	l.state = q.Get("appState")
	// When an expected nonce is armed (by Login), a mismatch is a stale or forged
	// callback: serve a failure page and report ErrStateMismatch. Empty wantState
	// disables the check (unit tests exercise the listener without one).
	if l.wantState != "" && l.state != l.wantState {
		l.cbErr = ErrStateMismatch
		writeResult(w, false, "Login failed. You may close this tab.")
		l.signal()
		return
	}
	writeResult(w, true, "Login complete. You may close this tab.")
	l.signal()
}

// signal closes l.received once (guard against duplicate close if a second
// request slips in before Shutdown takes effect).
func (l *Listener) signal() {
	select {
	case <-l.received:
	default:
		close(l.received)
	}
}

func writeResult(w http.ResponseWriter, ok bool, msg string) {
	status := http.StatusOK
	if !ok {
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	// Static page — never reflects query params (no XSS sink).
	_, _ = fmt.Fprintf(w, "<!doctype html><title>grn login</title><body style='font:15px sans-serif'><p>%s</p></body>", msg)
}
