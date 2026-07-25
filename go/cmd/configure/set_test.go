package configure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// displaySetValue must mask credentials so `set` never echoes a secret, while
// leaving non-sensitive values readable.
func TestDisplaySetValueMasksSecrets(t *testing.T) {
	secret := "super-secret-value-z789"
	for _, key := range []string{"client_id", "client_secret"} {
		got := displaySetValue(key, secret)
		if strings.Contains(got, "super-secret") {
			t.Errorf("%s: value not masked, got %q", key, got)
		}
		if !strings.HasSuffix(got, "z789") {
			t.Errorf("%s: expected masked value keeping last 4 chars, got %q", key, got)
		}
	}
	if got := displaySetValue("region", "HCM-3"); got != "HCM-3" {
		t.Errorf("region should not be masked, got %q", got)
	}
}

// newConfigureTestCmd wires a root command with the persistent `profile` flag
// (registered on rootCmd in production) so the configure subcommands resolve it
// exactly as they do at runtime.
func newConfigureTestCmd() *cobra.Command {
	root := &cobra.Command{Use: "grn"}
	root.PersistentFlags().String("profile", "", "")
	root.AddCommand(ConfigureCmd)
	return root
}

// Regression: `configure set <key> <value> --profile <new>` for a profile that
// does not exist in the credentials file must not panic. LoadConfig returns
// (nil, err) for an unknown profile; set previously dereferenced the nil *Config.
func TestSetRegionOnNonExistentProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := newConfigureTestCmd()
	root.SetArgs([]string{"configure", "set", "region", "HCM-3", "--profile", "prod-hcm-qc3"})
	if err := root.Execute(); err != nil {
		t.Fatalf("set region on new profile failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".greennode", "config"))
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "prod-hcm-qc3") || !strings.Contains(content, "HCM-3") {
		t.Errorf("config missing profile/region; got:\n%s", content)
	}
}

// On a fresh machine (no config files at all) `configure list` must not panic
// and renders unset defaults — matching `aws configure list`. (A profile that
// is missing while config files DO exist exits non-zero via os.Exit, which is
// covered by the binary-level checks rather than here.)
func TestListOnFreshMachineNoFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	root := newConfigureTestCmd()
	root.SetArgs([]string{"configure", "list", "--profile", "ghost"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list on fresh machine failed: %v", err)
	}
}
