package config

import "testing"

// TestDevEndpoints: the dev env resolves a full endpoint set, distinct from prod.
func TestDevEndpoints(t *testing.T) {
	eps := EndpointsForEnv(EnvDev)
	if eps.Identity == "" || eps.Runtime == "" || eps.Memory == "" || eps.Gateway == "" || eps.BuiltInTools == "" || eps.OAuth2Token == "" {
		t.Error("dev endpoints should not be empty")
	}
	if eps.Identity == EndpointsForEnv(EnvProd).Identity {
		t.Error("dev and prod identity endpoints should differ")
	}
	if got, want := eps.BuiltInTools, "https://pub-iamapis.api-dev.vngcloud.tech/built-in-tools"; got != want {
		t.Errorf("dev BuiltInTools = %q, want %q", got, want)
	}
}

// TestProdEndpoints: the prod env resolves a full endpoint set.
func TestProdEndpoints(t *testing.T) {
	eps := EndpointsForEnv(EnvProd)
	if eps.Identity == "" || eps.Runtime == "" || eps.Gateway == "" || eps.BuiltInTools == "" {
		t.Error("prod endpoints should not be empty")
	}
	if got, want := eps.BuiltInTools, "https://agentbase.api.vngcloud.vn/built-in-tools"; got != want {
		t.Errorf("prod BuiltInTools = %q, want %q", got, want)
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
