package login

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

// s256 mirrors the verifier→challenge transform the IAM leg uses, so the test
// is independent of the implementation under test (lifted from
// agent-core-gateway internal/handler/oauth/iam.go:66-67).
func s256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestGenerate_S256ChallengeMatches(t *testing.T) {
	t.Parallel()
	p, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got := s256(p.Verifier); got != p.Challenge {
		t.Errorf("S256(verifier)=%q, challenge=%q (must match)", got, p.Challenge)
	}
}

func TestGenerate_NonEmptyBase64URL(t *testing.T) {
	t.Parallel()
	p, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if p.Verifier == "" || p.Challenge == "" {
		t.Fatal("verifier and challenge must be non-empty")
	}
	// RFC 7636 §4.1: verifier is the unreserved set. base64.RawURLEncoding uses
	// [A-Za-z0-9_-], all unreserved — no padding. Reject '='.
	if strings.ContainsAny(p.Verifier, "+/=") {
		t.Errorf("verifier must be base64url (no +/=): %q", p.Verifier)
	}
}

func TestGenerate_Unique(t *testing.T) {
	t.Parallel()
	a, err := Generate()
	if err != nil {
		t.Fatalf("Generate a: %v", err)
	}
	seen := map[string]bool{a.Verifier: true}
	for i := 0; i < 32; i++ {
		p, err := Generate()
		if err != nil {
			t.Fatalf("Generate #%d: %v", i, err)
		}
		if seen[p.Verifier] {
			t.Fatalf("duplicate verifier at i=%d ( randomness failure)", i)
		}
		seen[p.Verifier] = true
	}
}
