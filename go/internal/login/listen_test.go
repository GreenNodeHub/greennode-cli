package login

import (
	"context"
	"errors"
	"io"
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

// TestServe_SuccessPageRendersUnifiedUI proves the local callback page mirrors
// the console's MCP login-callback UX: GreenNode brand, a success title, and a
// one-line "you can close this tab" instruction (no button — the tab is
// OS-launched, so window.close() cannot close it; see listen.go closeCaptions).
// Security-critical: no query-param value is reflected into the page (XSS guard
// — the page is static; Title/Close come from code constants only).
func TestServe_SuccessPageRendersUnifiedUI(t *testing.T) {
	t.Parallel()
	l, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer l.Close()

	done := make(chan struct{})
	go func() { l.Serve(context.Background()); close(done) }()

	// A distinctive code value we then assert is NOT reflected back.
	uri := l.RedirectURI() + "?code=SECRET-CODE-XYZ&appState=n"
	resp, err := http.Get(uri)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status=%d, want 200 (success page)", resp.StatusCode)
	}
	bs := string(body)
	// Console-faithful copy: the success TITLE is the console's i18n string,
	// not a local placeholder; the close instruction is "You can close this
	// tab."; the brand row is the GreenNode logo image alone (no wordmark text
	// — the wordmark lives in the logo). The success result-icon is the check
	// geometry (not the X).
	for _, want := range []string{"Authorization successful", "You can close this tab.", `result-icon__check`, `<img class="brand-mark"`, `data:image/png;base64,`, `alt="GreenNode"`} {
		if !strings.Contains(bs, want) {
			t.Errorf("success page missing %q; body: %s", want, bs)
		}
	}
	for _, notWant := range []string{"result-icon__bar--1", "brand-dot", ">GreenNode<", `<button`, `class="btn"`} {
		if strings.Contains(bs, notWant) {
			t.Errorf("success page unexpectedly contains %q; body: %s", notWant, bs)
		}
	}
	// The console button label must NOT appear — the local page has no button.
	if strings.Contains(bs, "Go to home") {
		t.Errorf("page still says 'Go to home'; want no button")
	}
	// XSS guard: query values must never be reflected into the static page.
	for _, secret := range []string{"SECRET-CODE-XYZ"} {
		if strings.Contains(bs, secret) {
			t.Errorf("page reflected query value %q (XSS sink); body: %s", secret, bs)
		}
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return within 2s")
	}
}

// The error/denied page shares the unified shell: red icon + the same "you can
// close this tab" instruction, and must not reflect the IAM error
// code/description from the query string.
func TestServe_DeniedPageDoesNotReflectErrorParams(t *testing.T) {
	t.Parallel()
	l, err := NewListener()
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer l.Close()

	done := make(chan struct{})
	go func() { l.Serve(context.Background()); close(done) }()

	uri := l.RedirectURI() + "?error=access_denied&error_description=<script>alert(1)</script>"
	resp, err := http.Get(uri)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (denied page)", resp.StatusCode)
	}
	bs := string(body)
	for _, want := range []string{"Authorization failed", "You can close this tab.", `result-icon__bar--1`} {
		if !strings.Contains(bs, want) {
			t.Errorf("denied page missing %q; body: %s", want, bs)
		}
	}
	// The success check must NOT appear on the denied page (it uses the X bars).
	if strings.Contains(bs, "result-icon__check") {
		t.Errorf("denied page shows success check; want X bars; body: %s", bs)
	}
	// The raw error code and the injected script payload must NOT appear.
	for _, bad := range []string{"access_denied", "<script>", "alert(1)"} {
		if strings.Contains(bs, bad) {
			t.Errorf("denied page reflected error param %q (XSS sink); body: %s", bad, bs)
		}
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return within 2s")
	}
}

// TestServe_LanguageFollowsAcceptLanguage proves the bilingual (en/vi) copy
// follows the request's Accept-Language: "vi" yields the Vietnamese title +
// "Bạn có thể đóng tab này." instruction (console-faithful); absence / "en"
// yields the English copy. The language is picked from a header the browser
// sends, not from any query param — so there is still no XSS surface.
func TestServe_LanguageFollowsAcceptLanguage(t *testing.T) {
	t.Parallel()
	fetch := func(t *testing.T, uri, acceptLang string) string {
		req, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		if acceptLang != "" {
			req.Header.Set("Accept-Language", acceptLang)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Do: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return string(body)
	}

	cases := []struct {
		name        string
		uri         string
		acceptLang  string
		wantLang    string // <html lang="..."> attribute
		wantTitle   string
		wantClose   string
		wantNotInEn bool // string that must NOT appear in the en variant
	}{
		{name: "success vi", uri: "?code=c&appState=n", acceptLang: "vi-VN,vi;q=0.9,en;q=0.8", wantLang: "vi", wantTitle: "Xác thực thành công", wantClose: "Bạn có thể đóng tab này."},
		{name: "success en default", uri: "?code=c&appState=n", acceptLang: "", wantLang: "en", wantTitle: "Authorization successful", wantClose: "You can close this tab.", wantNotInEn: true},
		{name: "success en header", uri: "?code=c&appState=n", acceptLang: "en-US,en;q=0.9", wantLang: "en", wantTitle: "Authorization successful", wantClose: "You can close this tab.", wantNotInEn: true},
		{name: "denied vi", uri: "?error=access_denied", acceptLang: "vi", wantLang: "vi", wantTitle: "Xác thực thất bại", wantClose: "Bạn có thể đóng tab này."},
		{name: "denied en header", uri: "?error=access_denied", acceptLang: "en", wantLang: "en", wantTitle: "Authorization failed", wantClose: "You can close this tab.", wantNotInEn: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l, err := NewListener()
			if err != nil {
				t.Fatalf("NewListener: %v", err)
			}
			defer l.Close()
			done := make(chan struct{})
			go func() { l.Serve(context.Background()); close(done) }()
			body := fetch(t, l.RedirectURI()+tc.uri, tc.acceptLang)
			if want := `lang="` + tc.wantLang + `"`; !strings.Contains(body, want) {
				t.Errorf("lang attr: missing %q; body head: %s", want, body[:min(220, len(body))])
			}
			if !strings.Contains(body, tc.wantTitle) {
				t.Errorf("missing title %q; body: %s", tc.wantTitle, body)
			}
			if !strings.Contains(body, `>`+tc.wantClose+`<`) {
				t.Errorf("missing close caption %q; body: %s", tc.wantClose, body)
			}
			if tc.wantNotInEn {
				for _, viOnly := range []string{"Xác thực", "Bạn có thể đóng tab này."} {
					if strings.Contains(body, viOnly) {
						t.Errorf("en variant leaked vi string %q; body: %s", viOnly, body)
					}
				}
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Serve did not return within 2s")
			}
		})
	}
}
