package login

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// Pair is a generated PKCE verifier and its S256 challenge.
type Pair struct {
	// Verifier is the code verifier sent to the token endpoint at exchange
	// time (RFC 7636). 256 bits of randomness, base64url-encoded (43 chars).
	Verifier string
	// Challenge is base64url(SHA256(Verifier)), sent on the authorize URL.
	Challenge string
}

// Generate returns a fresh S256 PKCE pair: a 256-bit random verifier
// (base64url) and base64url(SHA256(verifier)). Lifted from
// agent-core-gateway internal/handler/oauth/iam.go:61-67 (verifier = a 256-bit
// random base64url string, drawn via the same helper as the gateway's
// session.NewSID).
func Generate() (Pair, error) {
	var b [32]byte // 256 bits
	if _, err := rand.Read(b[:]); err != nil {
		return Pair{}, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(b[:])
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return Pair{Verifier: verifier, Challenge: challenge}, nil
}
