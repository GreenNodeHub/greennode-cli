package vks

import (
	"os"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func TestVKSResourceDryRunsAreOffline(t *testing.T) {
	cases := []struct {
		command *cobra.Command
		args    []string
	}{
		{deleteClusterCmd, []string{"--cluster-id", "fixture-cluster"}},
		{deleteNodegroupCmd, []string{"--cluster-id", "fixture-cluster", "--nodegroup-id", "fixture-nodegroup", "--force-delete"}},
		{updateKubeconfigCmd, []string{"--cluster-id", "fixture-cluster", "--alias", "fixture-context"}},
	}

	for _, tc := range cases {
		t.Run(tc.command.Name(), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			for _, name := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_ACCESS_KEY_ID", "GRN_SECRET_ACCESS_KEY", "GRN_PROFILE"} {
				t.Setenv(name, "")
			}

			root := mutationPortRoot(t, tc.command)
			root.SetArgs(append([]string{tc.command.Name()}, append(tc.args, "--dry-run")...))
			output := testutil.CaptureStdout(t, func() {
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
			})
			if !strings.Contains(output, "=== DRY RUN ===") {
				t.Fatalf("preview missing: %s", output)
			}
			entries, err := os.ReadDir(home)
			if err != nil || len(entries) != 0 {
				t.Fatal("dry-run wrote profile files")
			}
		})
	}
}
