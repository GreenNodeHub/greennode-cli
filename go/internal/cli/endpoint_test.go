package cli

import "testing"

func TestIsTrustedEndpoint(t *testing.T) {
	cases := map[string]bool{
		"":                                      true, // no override -> built-in endpoint
		"https://vks.api.vngcloud.vn":           true,
		"https://hcm-3.api.vngcloud.vn/x":       true,
		"https://vngcloud.vn":                   true,
		"https://api.greennode.ai":              true,
		"https://greennode.ai":                  true,
		"http://attacker.com":                   false,
		"https://evil.vngcloud.vn.attacker.com": false, // suffix must be a real domain boundary
		"https://notgreennode.ai":               false, // must match on a dot boundary
		"http://localhost:8080":                 false,
		"not-a-url ::::":                        false,
	}
	for in, want := range cases {
		if got := IsTrustedEndpoint(in); got != want {
			t.Errorf("IsTrustedEndpoint(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCheckEndpointPolicy(t *testing.T) {
	cases := []struct {
		name           string
		endpoint       string
		noVerifySSL    bool
		allowUntrusted bool
		wantBlocked    bool
	}{
		{"trusted https", "https://vks.api.vngcloud.vn", false, false, false},
		{"trusted greennode", "https://api.greennode.ai", false, false, false},
		{"no override", "", false, false, false},
		{"untrusted https verified -> warn only", "https://custom.example.com", false, false, false},
		{"untrusted http -> block", "http://attacker.com", false, false, true},
		{"untrusted https + no-verify -> block", "https://attacker.com", true, false, true},
		{"untrusted http + allow -> warn", "http://attacker.com", false, true, false},
		{"untrusted https + no-verify + allow -> warn", "https://attacker.com", true, true, false},
	}
	for _, tc := range cases {
		err := CheckEndpoint(tc.endpoint, tc.noVerifySSL, tc.allowUntrusted)
		if (err != nil) != tc.wantBlocked {
			t.Errorf("%s: blocked=%v, want %v (err=%v)", tc.name, err != nil, tc.wantBlocked, err)
		}
	}
}
