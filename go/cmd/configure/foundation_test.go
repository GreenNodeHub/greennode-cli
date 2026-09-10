package configure

import (
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
	"testing"
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
