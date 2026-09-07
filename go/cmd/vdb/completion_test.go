package vdb

import (
	"fmt"
	"sort"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// completableFlags are the flag names whose values come from the API or another vDB
// resource, so a user cannot reasonably type them from memory: ids, engine names,
// versions, zones. Every one of them must have a completion function bound, on every
// command that defines it.
//
// This test exists because the binding fails SILENTLY. RegisterFlagCompletionFunc
// returns an error when the flag does not exist yet, every call site discards it with
// //nolint:errcheck, and init() functions run in filename order — so a binding written in
// the wrong file compiles, ships, and simply does nothing. That happened once already
// (see the note in relational/instance/completion.go); walking the real command tree is
// the only way to notice.
var completableFlags = map[string]bool{
	"instance-id": true, "cluster-id": true, "config-id": true, "backup-id": true,
	"storage-id": true, "package-id": true, "zone-id": true, "subnet-ids": true,
	"datastore-type": true, "datastore-version": true, "volume-type": true,
	"status": true, "user-type": true,
	// Kafka's own resource flags. It has nouns no other product has (topics, users,
	// versioned config groups) and its security rules are addressed by ID.
	"topic-id": true, "user-id": true, "secrule-id": true, "flavor-id": true,
	"kafka-version": true, "config-group-id": true, "config-group-version-id": true,
	"network-id": true, "subnet-id": true, "from-version": true,
}

// completionExceptions are the flags deliberately left without value completion, each
// for the same reason: the flavors endpoint needs an engine and version, and on these
// commands neither is on the command line — they come from the instance or backup being
// acted on. Completing them would mean a second API call to resolve that resource first,
// plus giving the catalog package the other package's paths. Until that is judged worth
// it, they offer nothing rather than offering wrong values.
//
// Anything added here needs a reason. A flag that merely "was forgotten" belongs in the
// fix, not in this map.
var completionExceptions = map[string]string{
	"vdb relational instance resize-instance --package-id":  "engine and version come from the instance",
	"vdb relational instance create-replica --package-id":   "engine and version come from the source instance",
	"vdb relational backup restore --package-id":            "engine and version come from the backup",
	"vdb memorystore instance resize-instance --package-id": "version comes from the instance",
	"vdb memorystore instance create-replica --package-id":  "version comes from the source instance",
	"vdb memorystore backup restore --package-id":           "version comes from the backup",
}

func TestEveryCompletableFlagHasACompleter(t *testing.T) {
	var missing, staleException []string

	var walk func(cmd *cobra.Command, path string)
	walk = func(cmd *cobra.Command, path string) {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !completableFlags[f.Name] {
				return
			}
			key := fmt.Sprintf("%s --%s", path, f.Name)
			_, bound := cmd.GetFlagCompletionFunc(f.Name)
			_, excused := completionExceptions[key]

			switch {
			case !bound && !excused:
				missing = append(missing, key)
			case bound && excused:
				// The exception outlived its reason: someone wired the flag up. Drop the
				// entry so the map keeps describing reality.
				staleException = append(staleException, key)
			}
		})
		for _, sub := range cmd.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(VdbCmd, "vdb")

	sort.Strings(missing)
	for _, key := range missing {
		t.Errorf("no value completion bound for %q — bind it in the init() of the file that defines the flag", key)
	}
	sort.Strings(staleException)
	for _, key := range staleException {
		t.Errorf("%q now has a completer; remove it from completionExceptions", key)
	}
}
