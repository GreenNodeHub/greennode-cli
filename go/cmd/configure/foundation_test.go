package configure

import (
	"os"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func TestSetPortalUserIDReturnsErrorsAndPreservesProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_CLIENT_ID", "")
	t.Setenv("GRN_CLIENT_SECRET", "")
	t.Setenv("GRN_PORTAL_USER_ID", "")
	cmd := &cobra.Command{}
	cmd.Flags().String("profile", "target", "")
	if err := config.NewConfigFileWriter().WriteConfig("target", "HAN", "table", "project"); err != nil {
		t.Fatal(err)
	}
	if err := runSet(cmd, []string{"portal_user_id", "42"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig("target")
	if err != nil || cfg.PortalUserID != "42" || cfg.Output != "table" {
		t.Fatal("set lost profile configuration")
	}
	if err := runSet(cmd, []string{"portal_user_id", "uuid"}); err == nil {
		t.Fatal("invalid ID accepted")
	}
}

func TestSetDoesNotPersistCredentialEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, name := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_ACCESS_KEY_ID", "GRN_SECRET_ACCESS_KEY"} {
		t.Setenv(name, "")
	}
	writer := config.NewConfigFileWriter()
	if err := writer.WriteCredentials("default", "fixture-file-id", "fixture-file-secret"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRN_CLIENT_SECRET", "fixture-env-secret")

	cmd := &cobra.Command{}
	cmd.Flags().String("profile", "default", "")
	if err := runSet(cmd, []string{"client_id", "fixture-new-id"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadProfileFiles("default")
	if err != nil || cfg.ClientID != "fixture-new-id" || cfg.ClientSecret != "fixture-file-secret" {
		t.Fatalf("configure set persisted environment credentials: %#v, %v", cfg, err)
	}
}

func TestClientSecretPrompt(t *testing.T) {
	cli.SetNonInteractive(false)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString("fixture-prompt-secret\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = previous
		_ = reader.Close()
	})

	var secret string
	output := testutil.CaptureStdout(t, func() { secret, err = promptForClientSecret() })
	if err != nil || secret != "fixture-prompt-secret" {
		t.Fatalf("prompt result = %q, %v", secret, err)
	}
	if strings.Contains(output, secret) {
		t.Fatal("prompt echoed the secret")
	}
}

func TestClientSecretPromptRejectsNonInteractive(t *testing.T) {
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	if _, err := promptForClientSecret(); err == nil || !strings.Contains(err.Error(), "non-interactive") {
		t.Fatalf("error = %v", err)
	}
}
