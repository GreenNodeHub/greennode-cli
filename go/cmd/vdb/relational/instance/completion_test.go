package instance

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestFlagCompletionsAreRegistered guards a bug that already happened: the
// completions used to be bound from one function called in instance.go's init(),
// which runs before list.go and list_histories.go have defined their flags, so
// cobra rejected those bindings and discarded the error. Everything looked wired
// and no flag actually completed. Binding now lives beside each flag; this test
// fails if it drifts back.
func TestCompletionBindingSurvivesInitOrder(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		flag string
	}{
		{getCmd, "instance-id"},
		{listHistoriesCmd, "instance-id"},
		{listCmd, "status"},
	}

	for _, c := range cases {
		if c.cmd.Flags().Lookup(c.flag) == nil {
			t.Errorf("%s has no --%s flag", c.cmd.Name(), c.flag)
			continue
		}
		if _, ok := c.cmd.GetFlagCompletionFunc(c.flag); !ok {
			t.Errorf("%s --%s has no completion function registered", c.cmd.Name(), c.flag)
		}
	}
}
