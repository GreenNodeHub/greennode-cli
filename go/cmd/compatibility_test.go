package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestUpstreamCommandsRemainAvailable(t *testing.T) {
	data, err := os.ReadFile("testdata/upstream_commands.json")
	if err != nil {
		t.Fatal(err)
	}
	var baseline struct {
		CommonFlags []string `json:"common_flags"`
		Commands    []struct {
			Path  string   `json:"path"`
			Flags []string `json:"flags"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	if len(baseline.Commands) != 246 {
		t.Fatalf("baseline has %d commands, want 246", len(baseline.Commands))
	}
	for _, entry := range baseline.Commands {
		t.Run(entry.Path, func(t *testing.T) {
			command, args, err := rootCmd.Find(strings.Fields(entry.Path))
			if err != nil || len(args) != 0 {
				t.Fatalf("command missing: %v, remaining args %v", err, args)
			}
			if got := strings.TrimPrefix(command.CommandPath(), rootCmd.Name()+" "); got != entry.Path {
				t.Fatalf("resolved %q, want %q", got, entry.Path)
			}
			for _, flags := range [][]string{baseline.CommonFlags, entry.Flags} {
				for _, name := range flags {
					if command.Flags().Lookup(name) == nil && command.InheritedFlags().Lookup(name) == nil {
						t.Errorf("flag --%s removed", name)
					}
				}
			}
		})
	}
}
