package config

import "testing"

// TestDevEndpoints: the dev env resolves a full endpoint set, distinct from prod.
func TestDevEndpoints(t *testing.T) {
	eps := EndpointsForEnv(EnvDev)
	if eps.Identity == "" || eps.Runtime == "" || eps.Memory == "" || eps.OAuth2Token == "" {
		t.Error("dev endpoints should not be empty")
	}
	if eps.Identity == EndpointsForEnv(EnvProd).Identity {
		t.Error("dev and prod identity endpoints should differ")
	}
}

// TestProdEndpoints: the prod env resolves a full endpoint set.
func TestProdEndpoints(t *testing.T) {
	eps := EndpointsForEnv(EnvProd)
	if eps.Identity == "" || eps.Runtime == "" {
		t.Error("prod endpoints should not be empty")
	}
}

// TestEndpointsForEnv_UnknownFallsBackToProd: a malformed env never panics — it
// resolves to the prod default so a bad iam_env degrades safely.
func TestEndpointsForEnv_UnknownFallsBackToProd(t *testing.T) {
	got := EndpointsForEnv(Env("staging"))
	prod := EndpointsForEnv(EnvProd)
	if got != prod {
		t.Errorf("unknown env should fall back to prod; got %+v, want %+v", got, prod)
	}
}

// TestEnvFromString_Valid: dev/prod parse cleanly.
func TestEnvFromString_Valid(t *testing.T) {
	for _, s := range []string{"dev", "prod"} {
		env, err := EnvFromString(s)
		if err != nil {
			t.Errorf("EnvFromString(%q): unexpected error: %v", s, err)
		}
		if env != Env(s) {
			t.Errorf("EnvFromString(%q)=%q, want %q", s, env, s)
		}
	}
}

// TestEnvFromString_Invalid: anything other than dev/prod (including empty) is
// rejected, so `context switch` can never persist a bogus iam_env.
func TestEnvFromString_Invalid(t *testing.T) {
	for _, s := range []string{"", "staging", "DEV", "prod "} {
		if _, err := EnvFromString(s); err == nil {
			t.Errorf("EnvFromString(%q): expected error, got nil", s)
		}
	}
}
