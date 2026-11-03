package catalog

import (
	"strings"
	"testing"
)

// TestSimpleListsAreWired checks the declarative table actually produces usable
// commands: every spec becomes a subcommand, carries help text and a column set,
// and offers --zone-id exactly when it is meant to be filterable by zone.
func TestSimpleListsAreWired(t *testing.T) {
	byName := map[string]bool{}
	for _, sub := range CatalogCmd.Commands() {
		byName[sub.Name()] = true
	}

	for _, spec := range simpleLists {
		if !byName[spec.use] {
			t.Errorf("%q is declared but not registered under catalog", spec.use)
			continue
		}
		if spec.short == "" || spec.long == "" {
			t.Errorf("%q is missing help text", spec.use)
		}
		if len(spec.columns) == 0 {
			t.Errorf("%q has no columns; table output of a wide vdb payload is unreadable without them", spec.use)
		}
		if !strings.HasPrefix(spec.path, basePath) {
			t.Errorf("%q path = %q, want it under %q", spec.use, spec.path, basePath)
		}

		cmd, _, err := CatalogCmd.Find([]string{spec.use})
		if err != nil {
			t.Errorf("finding %q: %v", spec.use, err)
			continue
		}
		hasZone := cmd.Flags().Lookup("zone-id") != nil
		if hasZone != spec.byZone {
			t.Errorf("%q has --zone-id = %v, want %v", spec.use, hasZone, spec.byZone)
		}
		if !spec.byZone {
			continue
		}
		if _, ok := cmd.GetFlagCompletionFunc("zone-id"); !ok {
			t.Errorf("%q --zone-id has no completion function registered", spec.use)
		}
	}
}

// TestFlavorFlagsAndCompletions pins the flag names and their bindings. The
// binding check matters because RegisterFlagCompletionFunc fails silently when it
// runs before the flag exists — see the note in completion.go.
func TestFlavorFlagsAndCompletions(t *testing.T) {
	for _, flag := range []string{"datastore-type", "datastore-version", "zone-id"} {
		if listFlavorsCmd.Flags().Lookup(flag) == nil {
			t.Errorf("list-flavors has no --%s flag", flag)
			continue
		}
		if _, ok := listFlavorsCmd.GetFlagCompletionFunc(flag); !ok {
			t.Errorf("list-flavors --%s has no completion function registered", flag)
		}
	}

	// --version belongs to the root command; the datastore version must not
	// shadow it.
	if listFlavorsCmd.Flags().Lookup("version") != nil {
		t.Error("list-flavors defines --version, which collides with the global flag")
	}
}

func TestFlavorsQueryUsesAPIParamNames(t *testing.T) {
	defer func() {
		// listFlavorsCmd is package-level state shared with the command tree.
		_ = listFlavorsCmd.Flags().Set("datastore-type", "")
		_ = listFlavorsCmd.Flags().Set("datastore-version", "")
		_ = listFlavorsCmd.Flags().Set("zone-id", "")
	}()

	for flag, value := range map[string]string{
		"datastore-type": "postgresql", "datastore-version": "15", "zone-id": "HCM03-1A",
	} {
		if err := listFlavorsCmd.Flags().Set(flag, value); err != nil {
			t.Fatalf("setting --%s: %v", flag, err)
		}
	}

	got := flavorsQuery(listFlavorsCmd).Encode()
	want := "type=postgresql&version=15&zoneId=HCM03-1A"
	if got != want {
		t.Errorf("query = %q, want %q", got, want)
	}
}

// TestZoneQueryOmitsEmptyZone: an empty zoneId is not the same as "all zones" to
// every backend, so the param must be left out entirely when the flag is unset.
func TestZoneQueryOmitsEmptyZone(t *testing.T) {
	cmd, _, err := CatalogCmd.Find([]string{"list-volume-types"})
	if err != nil {
		t.Fatalf("finding list-volume-types: %v", err)
	}
	if got := zoneQuery(cmd).Encode(); got != "" {
		t.Errorf("query with no --zone-id = %q, want empty", got)
	}
}
