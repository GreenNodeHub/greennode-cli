package vdbclient

import (
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func newBuildClientTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("profile", "", "")
	cmd.Flags().String("region", "", "")
	cmd.Flags().String("endpoint-url", "", "")
	cmd.Flags().Bool("no-verify-ssl", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().Bool("allow-untrusted-endpoint", false, "")
	cmd.Flags().Int("cli-connect-timeout", 1, "")
	cmd.Flags().Int("cli-read-timeout", 1, "")
	return cmd
}

func configureEnvironment(t *testing.T, portalUserID string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_PROFILE", "fixture")
	t.Setenv("GRN_CLIENT_ID", "fixture-client-id")
	t.Setenv("GRN_CLIENT_SECRET", "fixture-client-secret")
	t.Setenv("GRN_PORTAL_USER_ID", portalUserID)
}

func TestValidatePortalUserID(t *testing.T) {
	for _, value := range []string{"1", "2147483647"} {
		if err := ValidatePortalUserID(value); err != nil {
			t.Errorf("ValidatePortalUserID(%q) error = %v", value, err)
		}
	}
	for _, value := range []string{"", "0", "-1", "abc", "2147483648"} {
		if err := ValidatePortalUserID(value); err == nil {
			t.Errorf("ValidatePortalUserID(%q) succeeded", value)
		}
	}
}

func TestBuildClientUsesGlobalEndpointWithoutRegion(t *testing.T) {
	configureEnvironment(t, "12345")
	apiClient, err := BuildClient(newBuildClientTestCommand(), true)
	if err != nil {
		t.Fatal(err)
	}
	if apiClient == nil {
		t.Fatal("BuildClient returned nil client")
	}
}

func TestBuildClientRejectsMissingOrInvalidPortalUserID(t *testing.T) {
	for _, portalUserID := range []string{"", "0", "-1", "abc", "2147483648"} {
		t.Run("portal-user-id="+portalUserID, func(t *testing.T) {
			configureEnvironment(t, portalUserID)
			_, err := BuildClient(newBuildClientTestCommand(), true)
			if err == nil || !strings.Contains(err.Error(), "portal_user_id") {
				t.Fatalf("BuildClient() error = %v, want portal_user_id validation", err)
			}
		})
	}
}

func TestBuildClientAllowsOmittedPortalUserIDForOperationsWithoutHeader(t *testing.T) {
	configureEnvironment(t, "")
	apiClient, err := BuildClient(newBuildClientTestCommand(), false)
	if err != nil {
		t.Fatal(err)
	}
	if apiClient == nil {
		t.Fatal("BuildClient returned nil client")
	}
}

func TestBuildClientRefusesUnprotectedUntrustedEndpoint(t *testing.T) {
	configureEnvironment(t, "12345")
	cmd := newBuildClientTestCommand()
	if err := cmd.Flags().Set("endpoint-url", "http://127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildClient(cmd, true); err == nil {
		t.Fatal("BuildClient accepted an untrusted plain HTTP endpoint")
	}
}

func TestBuildClientPrintsSSLWarningOnce(t *testing.T) {
	configureEnvironment(t, "12345")
	cmd := newBuildClientTestCommand()
	if err := cmd.Flags().Set("no-verify-ssl", "true"); err != nil {
		t.Fatal(err)
	}

	stderr := testutil.CaptureStderr(t, func() {
		if _, err := BuildClient(cmd, true); err != nil {
			t.Fatalf("BuildClient() error = %v", err)
		}
	})

	const warning = "Warning: SSL certificate verification is disabled. This is not recommended for production use."
	if got := strings.Count(stderr, warning); got != 1 {
		t.Fatalf("SSL warning printed %d time(s), want exactly 1; stderr = %q", got, stderr)
	}
}
