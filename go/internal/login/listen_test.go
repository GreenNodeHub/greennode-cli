package login

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListener_BindsLoopback_RandomPort(t *testing.T) {
	t.Parallel()
	l1, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer l1.Close()
	uri := l1.RedirectURI()
	if !strings.HasPrefix(uri, "http://127.0.0.1:") {
		t.Errorf("RedirectURI=%q, want http://127.0.0.1:<port>/callback", uri)
	}
	l2, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener 2: %v", err)
	}
	defer l2.Close()
	if l1.RedirectURI() == l2.RedirectURI() {
		t.Error("two listeners should bind different random ports")
	}
}

func TestServe_ReturnsCodeAndState(t *testing.T) {
	t.Parallel()
	l, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer l.Close()

	done := make(chan struct{})
	var code, state string
	var sErr error
	go func() {
		code, state, sErr = l.Serve(context.Background())
		close(done)
	}()

	// Give the goroutine a moment to call Accept, then GET the callback.
	uri := l.RedirectURI() + "?code=the-code&appState=the-nonce"
	resp, err := http.Get(uri)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	resp.Body.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return within 2s")
	}
	if sErr != nil {
		t.Fatalf("Serve err: %v", sErr)
	}
	if code != "the-code" {
		t.Errorf("code=%q, want the-code", code)
	}
	if state != "the-nonce" {
		t.Errorf("state=%q, want the-nonce", state)
	}
}

func TestServe_ErrorParamsReturnErrAuthzDenied(t *testing.T) {
	t.Parallel()
	l, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer l.Close()

	done := make(chan struct{})
	var sErr error
	go func() {
		_, _, sErr = l.Serve(context.Background())
		close(done)
	}()

	uri := l.RedirectURI() + "?error=access_denied&error_description=user+cancelled"
	resp, err := http.Get(uri)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	resp.Body.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return within 2s")
	}
	denied, ok := sErr.(*ErrAuthzDenied)
	if !ok {
		t.Fatalf("sErr=%v, want *ErrAuthzDenied", sErr)
	}
	if denied.Code != "access_denied" {
		t.Errorf("Code=%q, want access_denied", denied.Code)
	}
}

func TestServe_ContextCancelShutsDown(t *testing.T) {
	t.Parallel()
	l, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer l.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var sErr error
	go func() {
		_, _, sErr = l.Serve(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not shut down within 2s of ctx cancel")
	}
	if !errors.Is(sErr, context.Canceled) {
		t.Errorf("Serve cancel err=%v, want context.Canceled", sErr)
	}
}

func TestServe_StateMismatchServesFailurePage(t *testing.T) {
	t.Parallel()
	l, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer l.Close()
	l.setExpectedState("the-nonce") // arm the nonce check (as Login does)

	done := make(chan struct{})
	var sErr error
	go func() {
		_, _, sErr = l.Serve(context.Background())
		close(done)
	}()

	// Callback carries a DIFFERENT appState than the armed nonce.
	uri := l.RedirectURI() + "?code=c&appState=WRONG"
	resp, err := http.Get(uri)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return within 2s")
	}
	if !errors.Is(sErr, ErrStateMismatch) {
		t.Errorf("Serve err=%v, want ErrStateMismatch", sErr)
	}
	// The honest UX: a mismatch shows a FAILURE page, not "Login complete".
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d, want %d (failure page on state mismatch)", resp.StatusCode, http.StatusBadRequest)
	}
}
