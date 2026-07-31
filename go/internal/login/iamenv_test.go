package login

import (
	"strings"
	"testing"
)

func TestTokenURLForEnv_ProdAndDev(t *testing.T) {
	t.Parallel()
	cases := []struct {
		env  string
		want string
	}{
		{"prod", "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token"},
		{"dev", "https://pub-iamapis.api-dev.vngcloud.tech/accounts-api/v2/auth/token"},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Parallel()
			got, err := TokenURLForEnv(tc.env)
			if err != nil {
				t.Fatalf("TokenURLForEnv(%q): %v", tc.env, err)
			}
			if got != tc.want {
				t.Errorf("TokenURLForEnv(%q)=%q, want %q", tc.env, got, tc.want)
			}
		})
	}
}

func TestTokenURLForEnv_EmptyAndUnknownError(t *testing.T) {
	t.Parallel()
	for _, env := range []string{"", "staging", "prod "} {
		_, err := TokenURLForEnv(env)
		if err == nil {
			t.Errorf("TokenURLForEnv(%q) succeeded, want error", env)
			continue
		}
		// Empty env gets a "run `grn login`" hint; unknown gets the valid list.
		if env == "" && !strings.Contains(err.Error(), "grn login") {
			t.Errorf("empty-env err=%q, want guidance mentioning `grn login`", err)
		}
	}
}

func TestIamEndpoints_BakedClientIDs(t *testing.T) {
	t.Parallel()
	if got := IamEndpoints["dev"].ClientID; got != DevClientID {
		t.Errorf("dev ClientID=%q, want DevClientID %q", got, DevClientID)
	}
	if got := IamEndpoints["prod"].ClientID; got != ProdClientID {
		t.Errorf("prod ClientID=%q, want ProdClientID %q", got, ProdClientID)
	}
	// The dev token URL must differ from the prod v2 URL (and from the v1 machine
	// endpoint) — a mix-up would send refreshes to the wrong host.
	if IamEndpoints["dev"].Token == IamEndpoints["prod"].Token {
		t.Error("dev and prod share a token URL; expected distinct per-env endpoints")
	}
}

func TestEndpointForEnv(t *testing.T) {
	t.Parallel()
	ep, err := EndpointForEnv("dev")
	if err != nil {
		t.Fatalf("EndpointForEnv(dev): %v", err)
	}
	if ep.Authorize == "" || ep.Token == "" || ep.ClientID == "" {
		t.Errorf("dev preset incomplete: %+v", ep)
	}
	if _, err := EndpointForEnv("nope"); err == nil {
		t.Error("EndpointForEnv(nope) succeeded, want error")
	}
	if _, err := EndpointForEnv(""); err == nil {
		t.Error("EndpointForEnv() succeeded, want error")
	}
}
