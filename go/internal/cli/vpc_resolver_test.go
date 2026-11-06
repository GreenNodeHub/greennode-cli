package cli

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveClusterVPC(t *testing.T) {
	t.Cleanup(func() { RegisterClusterVPCResolver(nil) })

	// No resolver registered: "" with no error, so completion just yields nothing.
	if got, err := ResolveClusterVPC(&cobra.Command{}, "k8s-aaa"); got != "" || err != nil {
		t.Fatalf("unregistered: got (%q, %v), want (\"\", nil)", got, err)
	}

	var seen string
	RegisterClusterVPCResolver(func(_ *cobra.Command, clusterID string) (string, error) {
		seen = clusterID
		return "net-aaa", nil
	})
	got, err := ResolveClusterVPC(&cobra.Command{}, "k8s-aaa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "net-aaa" || seen != "k8s-aaa" {
		t.Errorf("got %q for cluster %q, want net-aaa for k8s-aaa", got, seen)
	}

	// An empty cluster ID must not reach the resolver.
	seen = ""
	if got, err := ResolveClusterVPC(&cobra.Command{}, ""); got != "" || err != nil || seen != "" {
		t.Errorf("empty cluster id: got (%q, %v), resolver saw %q", got, err, seen)
	}

	wantErr := errors.New("boom")
	RegisterClusterVPCResolver(func(_ *cobra.Command, _ string) (string, error) {
		return "", wantErr
	})
	if _, err := ResolveClusterVPC(&cobra.Command{}, "k8s-aaa"); !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v propagated to the caller", err, wantErr)
	}
}
