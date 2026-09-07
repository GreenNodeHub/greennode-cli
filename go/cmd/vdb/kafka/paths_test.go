package kafka

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestGroupIsComplete pins the five nouns and every command under them, so a dropped
// registration shows up here rather than as a missing command.
//
// 36 commands cover 35 of the API's 36 Kafka endpoints. The two numbers differ for
// two deliberate reasons, both asserted below:
//
//   - `cluster list-secrules` has NO endpoint. Kafka never lists security rules on
//     their own; they arrive nested in the cluster object, and their IDs are needed
//     for delete-secrule, so the command reads the cluster and prints them.
//   - `GET /vdb-kafka/database/configs` has NO command. It returns the service's own
//     application settings as a map<string,string> — not something a user acts on.
func TestGroupIsComplete(t *testing.T) {
	want := map[string][]string{
		"cluster": {
			"list", "get", "list-histories", "list-secrules",
			"create", "delete",
			"resize-brokers", "resize-storage", "update-volume-type",
			"update-authentication", "update-config-group", "update-public-access",
			"create-secrule", "delete-secrule",
		},
		"topic": {"list", "get", "create", "update", "delete"},
		"user":  {"list", "get", "get-creds", "create", "update", "delete", "generate-creds"},
		"configuration": {
			"list", "get", "get-version", "create", "create-version", "delete",
		},
		"catalog": {"list-families", "list-flavor-codes", "list-flavors", "list-volume-types"},
	}

	for noun, commands := range want {
		group, _, err := KafkaCmd.Find([]string{noun})
		if err != nil || group.Name() != noun {
			t.Errorf("noun %q is not registered", noun)
			continue
		}
		present := map[string]bool{}
		for _, sub := range group.Commands() {
			present[sub.Name()] = true
		}
		for _, name := range commands {
			if !present[name] {
				t.Errorf("kafka %s %s is missing", noun, name)
			}
		}
	}
}

// TestNoConfigsCommand: /vdb-kafka/database/configs returns the service's own
// application settings, which no user acts on. It must not gain a command by someone
// mechanically walking the spec.
func TestNoConfigsCommand(t *testing.T) {
	catalog, _, err := KafkaCmd.Find([]string{"catalog"})
	if err != nil {
		t.Fatal("catalog noun is not registered")
	}
	for _, sub := range catalog.Commands() {
		if strings.Contains(sub.Name(), "config") {
			t.Errorf("catalog %s exists; /vdb-kafka/database/configs is service configuration, not a user lookup",
				sub.Name())
		}
	}
}

// TestNoBackupNoun: Kafka is the one vDB product with no backup service. A backup
// noun here would mean someone copied it from another group.
func TestNoBackupNoun(t *testing.T) {
	for _, sub := range KafkaCmd.Commands() {
		name := sub.Name()
		if strings.Contains(name, "backup") {
			t.Errorf("kafka %s exists, but the Kafka API has no backup endpoints at all", name)
		}
	}
}

// TestEveryLeafHasHelp mirrors the repo-wide convention test, scoped to this group so
// a new command cannot land without a Short.
func TestEveryLeafHasHelp(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Runnable() && len(sub.Commands()) == 0 && strings.TrimSpace(sub.Short) == "" {
				t.Errorf("%s has no Short description", sub.CommandPath())
			}
			walk(sub)
		}
	}
	walk(KafkaCmd)
}

// TestMutatingCommandsAreGated goes beyond the repo-wide rule, which only covers
// delete/stop/reboot. Every Kafka command that places a paid order, changes who can
// reach the cluster, or replaces a set of values wholesale must offer --dry-run and
// --force too.
func TestMutatingCommandsAreGated(t *testing.T) {
	gated := map[string][]string{
		"cluster": {
			"create", "delete", "resize-brokers", "resize-storage", "update-volume-type",
			"update-authentication", "update-config-group", "update-public-access",
			"create-secrule", "delete-secrule",
		},
		"topic":         {"update", "delete"},
		"user":          {"update", "delete", "generate-creds"},
		"configuration": {"create-version", "delete"},
	}

	for noun, commands := range gated {
		for _, name := range commands {
			cmd, _, err := KafkaCmd.Find([]string{noun, name})
			if err != nil || cmd.Name() != name {
				t.Errorf("kafka %s %s is not registered", noun, name)
				continue
			}
			for _, flag := range []string{"dry-run", "force"} {
				if cmd.Flags().Lookup(flag) == nil {
					t.Errorf("kafka %s %s must define --%s", noun, name, flag)
				}
			}
		}
	}
}
