package iam

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExcludedIAMOperationsHaveNoCommand(t *testing.T) {
	gated := excludedIAMKeys()
	if len(gated) == 0 {
		t.Fatal("excludedOperations() is empty; the credential-issuance gate is gone")
	}

	for _, op := range allIAMMutationOperations() {
		if reason, ok := gated[op.Method+" "+op.Path]; ok {
			t.Errorf("%s %s ships as %q but is excluded (%s)", op.Method, op.Path, op.Key, reason)
		}
	}

	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			walk(child)
		}
		if cmd.RunE == nil {
			return
		}
		for _, banned := range []string{"issue-token", "impersonate"} {
			if cmd.Name() == banned {
				t.Errorf("command %q is mounted but issues live credentials", cmd.CommandPath())
			}
		}
	}
	walk(IamCmd)

	if command, _, err := IamCmd.Find([]string{"auth"}); err == nil && command != nil && command.Name() == "auth" {
		t.Error("iam auth is still mounted; IAM management must not add a separate authentication entry point")
	}
}

func TestExcludedIAMOperationsStateACredentialReason(t *testing.T) {
	for _, excluded := range excludedOperations() {
		if !strings.HasPrefix(excluded.Reason, reasonCredential) {
			t.Errorf("%s %s reason = %q, want the credential-material category", excluded.Method, excluded.Path, excluded.Reason)
		}
	}
}
