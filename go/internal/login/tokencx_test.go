package login

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// recServer records the last request it received (method, path, body, headers).
type recServer struct {
	body   string
	header http.Header
	status int
	resp   string
}

func newRecServer(resp string, status int) (*httptest.Server, *recServer) {
	r := &recServer{resp: resp, status: status}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		r.body = string(b)
		r.header = req.Header
		if r.status == 0 {
			r.status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(r.status)
		_, _ = w.Write([]byte(r.resp))
	}))
	return s, r
}

func TestExchangeCode_SendsCodeVerifierAndForm(t *testing.T) {
	t.Parallel()
	s, r := newRecServer(`{"access_token":"at","refresh_token":"rt","expires_in":3600}`, 0)
	defer s.Close()
	c := New(5 * time.Second)

	resp, errE, err := c.ExchangeCode(context.Background(), s.URL, ExchangeParams{
		Code: "the-code", RedirectURI: "http://127.0.0.1:1/callback",
		ClientID: "cid", CodeVerifier: "the-verifier",
	})
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if errE != nil {
		t.Fatalf("unexpected *Error: %+v", errE)
	}
	v, _ := url.ParseQuery(r.body)
	if v.Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type=%q, want authorization_code", v.Get("grant_type"))
	}
	if v.Get("code") != "the-code" {
		t.Errorf("code=%q", v.Get("code"))
	}
	if v.Get("code_verifier") != "the-verifier" {
		t.Errorf("code_verifier=%q, want the-verifier", v.Get("code_verifier"))
	}
	if v.Get("redirect_uri") != "http://127.0.0.1:1/callback" {
		t.Errorf("redirect_uri=%q", v.Get("redirect_uri"))
	}
	if v.Get("client_id") != "cid" {
		t.Errorf("client_id=%q", v.Get("client_id"))
	}
	if r.header.Get("Authorization") != "" {
		t.Errorf("no Basic expected without secret; got %q", r.header.Get("Authorization"))
	}
	if string(resp.Raw) == "" || !strings.Contains(string(resp.Raw), "access_token") {
		t.Errorf("Raw not preserved: %s", string(resp.Raw))
	}
}

func TestExchangeCode_BasicAuthWhenSecretSet(t *testing.T) {
	t.Parallel()
	s, r := newRecServer(`{"access_token":"at"}`, 0)
	defer s.Close()
	c := New(5 * time.Second)

	_, _, err := c.ExchangeCode(context.Background(), s.URL, ExchangeParams{
		Code: "c", RedirectURI: "u", ClientID: "cid", ClientSecret: "sec", CodeVerifier: "v",
	})
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	auth := r.header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("expected Basic auth when secret set; got %q", auth)
	}
	// client_secret must NOT also be in the form body (Basic is the auth).
	if v, _ := url.ParseQuery(r.body); v.Get("client_secret") != "" {
		t.Errorf("client_secret must not be in body when Basic is used; got %q", v.Get("client_secret"))
	}
}

func TestExchangeCode_Non2xxReturnsError(t *testing.T) {
	t.Parallel()
	s, _ := newRecServer(`{"error":"invalid_grant"}`, http.StatusBadRequest)
	defer s.Close()
	c := New(5 * time.Second)

	resp, errE, err := c.ExchangeCode(context.Background(), s.URL, ExchangeParams{
		Code: "c", RedirectURI: "u", ClientID: "cid", CodeVerifier: "v",
	})
	if err != nil {
		t.Fatalf("ExchangeCode transport err: %v", err)
	}
	if errE == nil {
		t.Fatalf("expected *Error for 400, got resp=%s", string(resp.Raw))
	}
	if errE.Status != http.StatusBadRequest {
		t.Errorf("errE.Status=%d, want 400", errE.Status)
	}
	if !strings.Contains(string(errE.RawBody), "invalid_grant") {
		t.Errorf("RawBody not preserved: %s", string(errE.RawBody))
	}
}

func TestRefresh_SendsRefreshToken(t *testing.T) {
	t.Parallel()
	s, r := newRecServer(`{"access_token":"at2","refresh_token":"rt2","expires_in":3600}`, 0)
	defer s.Close()
	c := New(5 * time.Second)

	_, _, err := c.Refresh(context.Background(), s.URL, RefreshParams{
		RefreshToken: "rt", ClientID: "cid", ClientSecret: "sec", Scope: "openid",
	})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	v, _ := url.ParseQuery(r.body)
	if v.Get("grant_type") != "refresh_token" {
		t.Errorf("grant_type=%q, want refresh_token", v.Get("grant_type"))
	}
	if v.Get("refresh_token") != "rt" {
		t.Errorf("refresh_token=%q", v.Get("refresh_token"))
	}
	if v.Get("scope") != "openid" {
		t.Errorf("scope=%q", v.Get("scope"))
	}
	if r.header.Get("Authorization") == "" {
		t.Error("expected Basic auth with secret set")
	}
}
