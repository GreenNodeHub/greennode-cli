package iamclient

import (
	"reflect"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

func clientTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	flags := cmd.Flags()
	flags.String("profile", "", "")
	flags.String("region", "", "")
	flags.String("endpoint-url", "", "")
	flags.Bool("no-verify-ssl", false, "")
	flags.Bool("debug", false, "")
	flags.Bool("allow-untrusted-endpoint", false, "")
	flags.Int("cli-connect-timeout", 0, "")
	flags.Int("cli-read-timeout", 0, "")
	return cmd
}

func TestBuildClientsUseDocumentedEndpoints(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_PROFILE", "")
	t.Setenv("GRN_CLIENT_ID", "test-client-id")
	t.Setenv("GRN_CLIENT_SECRET", "test-client-secret")

	tests := []struct {
		name     string
		build    func(*cobra.Command) (*client.GreennodeClient, error)
		endpoint string
	}{
		{name: "accounts", build: BuildAccountsClient, endpoint: AccountsEndpoint},
		{name: "policies", build: BuildPoliciesClient, endpoint: PoliciesEndpoint},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			greennodeClient, err := tc.build(clientTestCommand())
			if err != nil {
				t.Fatalf("build client: %v", err)
			}
			baseURL := reflect.ValueOf(greennodeClient).Elem().FieldByName("baseURL")
			if !baseURL.IsValid() || baseURL.Kind() != reflect.String {
				t.Fatal("GreennodeClient base URL is unavailable")
			}
			if got := baseURL.String(); got != tc.endpoint {
				t.Errorf("client base URL = %q, want %q", got, tc.endpoint)
			}
		})
	}
}
