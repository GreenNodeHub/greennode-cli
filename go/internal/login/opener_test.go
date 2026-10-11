package login

import "testing"

func TestQuoteWindowsURLPreservesQuerySeparators(t *testing.T) {
	rawURL := "https://example.com/auth?appState=abc&clientId=def&codeChallenge=ghi"

	quoted, err := quoteWindowsURL(rawURL)
	if err != nil {
		t.Fatalf("quoteWindowsURL() error = %v", err)
	}
	if quoted != `"`+rawURL+`"` {
		t.Fatalf("quoteWindowsURL() = %q, want %q", quoted, `"`+rawURL+`"`)
	}
}

func TestQuoteWindowsURLRejectsEmbeddedQuotes(t *testing.T) {
	if _, err := quoteWindowsURL(`https://example.com/auth?next="bad"`); err == nil {
		t.Fatal("quoteWindowsURL() error = nil, want error")
	}
}
