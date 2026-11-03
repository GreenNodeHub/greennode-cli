package configuration

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireConfigIDAcceptsBothProducts(t *testing.T) {
	// The endpoint family serves single-instance and cluster groups alike, so both
	// prefixes are valid; only an ID from another resource is not.
	for _, id := range []string{"cfg-b8f60c5c-e96a", "pg-cfg-b8cf4290-9c11"} {
		if err := requireConfigID(id); err != nil {
			t.Errorf("requireConfigID(%q) = %v, want nil", id, err)
		}
	}
	for _, id := range []string{"db-ceb5fd7b", "bk-740f7917", "", "cfg id with spaces"} {
		if err := requireConfigID(id); err == nil {
			t.Errorf("requireConfigID(%q) = nil, want an error", id)
		}
	}
}

// TestParseValuesTypesTheValues: the API's parameter definitions are typed and its
// own example sends {"autocommit": 1} — a number. Sending "1" leaves the caller at
// the mercy of the backend's coercion.
func TestParseValuesTypesTheValues(t *testing.T) {
	values, err := parseValues([]string{
		"autocommit=1",
		"long_query_time=2.5",
		"general_log=true",
		"sql_mode=STRICT_TRANS_TABLES",
	})
	if err != nil {
		t.Fatalf("parseValues: %v", err)
	}

	if got, ok := values["autocommit"].(int64); !ok || got != 1 {
		t.Errorf("autocommit = %#v, want int64(1)", values["autocommit"])
	}
	if got, ok := values["long_query_time"].(float64); !ok || got != 2.5 {
		t.Errorf("long_query_time = %#v, want float64(2.5)", values["long_query_time"])
	}
	if got, ok := values["general_log"].(bool); !ok || !got {
		t.Errorf("general_log = %#v, want true", values["general_log"])
	}
	if got, ok := values["sql_mode"].(string); !ok || got != "STRICT_TRANS_TABLES" {
		t.Errorf("sql_mode = %#v, want the string", values["sql_mode"])
	}
}

func TestParseValuesRejectsBadInput(t *testing.T) {
	for name, specs := range map[string][]string{
		"no equals sign":  {"autocommit"},
		"empty name":      {"=1"},
		"nothing at all":  {},
		"same name twice": {"autocommit=1", "autocommit=0"},
	} {
		if _, err := parseValues(specs); err == nil {
			t.Errorf("%s: parseValues(%v) = nil error, want a rejection", name, specs)
		}
	}
}

// TestDescribeValuesKeepsNumbersReadable: parameter values are float64 after JSON
// decoding, and %v renders 8388608 as 8.388608e+06.
func TestDescribeValuesKeepsNumbersReadable(t *testing.T) {
	got := describeValues(map[string]interface{}{
		"bulk_insert_buffer_size": float64(8388608),
		"autocommit":              float64(1),
		"sql_mode":                "STRICT",
	})

	if strings.Contains(got, "e+") {
		t.Errorf("describeValues used exponent form: %s", got)
	}
	// Sorted, so two runs can be compared by eye.
	want := "autocommit=1, bulk_insert_buffer_size=8388608, sql_mode=STRICT"
	if got != want {
		t.Errorf("describeValues = %q, want %q", got, want)
	}
	if describeValues(nil) != "(none)" {
		t.Error("an empty parameter set should read as (none)")
	}
}

// TestUpdateOffersMerge: the API replaces the whole parameter set, so a way to keep
// the existing parameters is not optional.
func TestUpdateOffersMerge(t *testing.T) {
	if updateCmd.Flags().Lookup("merge") == nil {
		t.Error("configuration update must offer --merge; the API replaces the whole set")
	}
	for _, flag := range []string{"dry-run", "force"} {
		if updateCmd.Flags().Lookup(flag) == nil {
			t.Errorf("configuration update must define --%s", flag)
		}
	}
}

func TestCommandsAreWired(t *testing.T) {
	want := map[string]bool{
		"list": true, "get": true, "create": true,
		"update": true, "delete": true, "list-params": true,
	}
	for _, sub := range ConfigurationCmd.Commands() {
		delete(want, sub.Name())
	}
	for name := range want {
		t.Errorf("configuration %s is not registered", name)
	}
}

func TestFlagCompletionsAreRegistered(t *testing.T) {
	cases := []struct {
		cmd   *cobra.Command
		flags []string
	}{
		{getCmd, []string{"config-id"}},
		{updateCmd, []string{"config-id"}},
		{deleteCmd, []string{"config-id"}},
		{createCmd, []string{"datastore-type", "datastore-version", "deploy-type"}},
		{listParamsCmd, []string{"datastore-type", "datastore-version", "deploy-type"}},
	}

	for _, c := range cases {
		for _, flag := range c.flags {
			if c.cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%s has no --%s flag", c.cmd.Name(), flag)
				continue
			}
			if _, ok := c.cmd.GetFlagCompletionFunc(flag); !ok {
				t.Errorf("%s --%s has no completion function registered", c.cmd.Name(), flag)
			}
		}
	}
}
