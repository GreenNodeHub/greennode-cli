package operation

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newQueryTestCommand(params []QueryParam) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	for _, p := range params {
		cmd.Flags().String(p.Flag, "", p.Usage)
	}
	return cmd
}

func TestBuildQueryElidesEmptyResultToNilMap(t *testing.T) {
	params := []QueryParam{{WireName: "name", Flag: "name"}}
	cmd := newQueryTestCommand(params)
	query, err := BuildQuery(cmd, params)
	if err != nil {
		t.Fatalf("BuildQuery() error = %v", err)
	}
	if query != nil {
		t.Fatalf("query = %#v, want nil for an all-empty optional parameter set", query)
	}
}

func TestBuildQueryWithNoParametersReturnsNil(t *testing.T) {
	query, err := BuildQuery(&cobra.Command{Use: "test"}, nil)
	if err != nil || query != nil {
		t.Fatalf("BuildQuery() = (%#v, %v), want (nil, nil)", query, err)
	}
}

func TestBuildQueryValidateIDGatesOnlyMarkedParameters(t *testing.T) {
	params := []QueryParam{
		{WireName: "id", Flag: "id", Kind: QueryID},
		{WireName: "note", Flag: "note"},
	}
	cmd := newQueryTestCommand(params)
	if err := cmd.Flags().Set("id", "bad id with spaces"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("note", "bad id with spaces too"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildQuery(cmd, params); err == nil {
		t.Fatal("BuildQuery() error = nil, want validator.ValidateID rejection of --id")
	}

	cmd2 := newQueryTestCommand(params)
	if err := cmd2.Flags().Set("note", "bad id with spaces too"); err != nil {
		t.Fatal(err)
	}
	query, err := BuildQuery(cmd2, params)
	if err != nil {
		t.Fatalf("BuildQuery() error = %v, want unvalidated note parameter accepted", err)
	}
	if query["note"] != "bad id with spaces too" {
		t.Fatalf("query = %#v, want note passed through unchanged", query)
	}
}

func TestBuildQueryRequiredEmptyFails(t *testing.T) {
	params := []QueryParam{{WireName: "name", Flag: "name", Required: true}}
	cmd := newQueryTestCommand(params)
	if _, err := BuildQuery(cmd, params); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("BuildQuery() error = %v, want required-empty rejection", err)
	}
}

func TestBuildQueryDefaultFillsEmptyValueBeforeRequiredCheck(t *testing.T) {
	params := []QueryParam{{WireName: "page", Flag: "page", Required: true, Default: "0"}}
	cmd := newQueryTestCommand(params)
	query, err := BuildQuery(cmd, params)
	if err != nil {
		t.Fatalf("BuildQuery() error = %v, want Default to satisfy Required", err)
	}
	if query["page"] != "0" {
		t.Fatalf("query = %#v, want page defaulted to 0", query)
	}
}

func TestBuildQueryIntegerKindValidatesAndBoundsValue(t *testing.T) {
	params := []QueryParam{{WireName: "size", Flag: "size", Kind: QueryInteger, Minimum: 1}}

	cmd := newQueryTestCommand(params)
	if err := cmd.Flags().Set("size", "not-a-number"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildQuery(cmd, params); err == nil {
		t.Fatal("BuildQuery() error = nil, want non-integer rejection")
	}

	cmd = newQueryTestCommand(params)
	if err := cmd.Flags().Set("size", "0"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildQuery(cmd, params); err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("BuildQuery() error = %v, want minimum-bound rejection", err)
	}

	cmd = newQueryTestCommand(params)
	if err := cmd.Flags().Set("size", "5"); err != nil {
		t.Fatal(err)
	}
	query, err := BuildQuery(cmd, params)
	if err != nil || query["size"] != "5" {
		t.Fatalf("BuildQuery() = (%#v, %v), want size=5 accepted", query, err)
	}
}

func TestBuildQueryBooleanKindNormalizesValue(t *testing.T) {
	params := []QueryParam{{WireName: "enabled", Flag: "enabled", Kind: QueryBoolean}}
	cmd := newQueryTestCommand(params)
	if err := cmd.Flags().Set("enabled", "1"); err != nil {
		t.Fatal(err)
	}
	query, err := BuildQuery(cmd, params)
	if err != nil {
		t.Fatalf("BuildQuery() error = %v", err)
	}
	if query["enabled"] != "true" {
		t.Fatalf("query[enabled] = %q, want normalized \"true\"", query["enabled"])
	}
}

func TestBuildQueryObjectKindRequiresJSONObjectAndReencodesCompactly(t *testing.T) {
	params := []QueryParam{{WireName: "filter", Flag: "filter", Kind: QueryObject}}

	cmd := newQueryTestCommand(params)
	if err := cmd.Flags().Set("filter", "not json"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildQuery(cmd, params); err == nil {
		t.Fatal("BuildQuery() error = nil, want invalid JSON rejection")
	}

	cmd = newQueryTestCommand(params)
	if err := cmd.Flags().Set("filter", `[]`); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildQuery(cmd, params); err == nil {
		t.Fatal("BuildQuery() error = nil, want non-object JSON rejection")
	}

	cmd = newQueryTestCommand(params)
	if err := cmd.Flags().Set("filter", `{  "b" : 2 , "a" : 1 }`); err != nil {
		t.Fatal(err)
	}
	query, err := BuildQuery(cmd, params)
	if err != nil {
		t.Fatalf("BuildQuery() error = %v", err)
	}
	if query["filter"] != `{"a":1,"b":2}` {
		t.Fatalf("query[filter] = %q, want compact re-encoded object", query["filter"])
	}
}

func TestBuildQueryObjectKindUsesSelectedStyleForDecodeError(t *testing.T) {
	defaultStyle := []QueryParam{{WireName: "filter", Flag: "filter", Kind: QueryObject}}
	cmd := newQueryTestCommand(defaultStyle)
	if err := cmd.Flags().Set("filter", "not json"); err != nil {
		t.Fatal(err)
	}
	_, err := BuildQuery(cmd, defaultStyle)
	if err == nil || !strings.Contains(err.Error(), "invalid filter JSON object:") {
		t.Fatalf("BuildQuery() error = %v, want default QueryObjectJSONObject wording", err)
	}

	jsonStyle := []QueryParam{{WireName: "filter", Flag: "filter", Kind: QueryObject, ObjectStyle: QueryObjectJSON}}
	cmd = newQueryTestCommand(jsonStyle)
	if err := cmd.Flags().Set("filter", "not json"); err != nil {
		t.Fatal(err)
	}
	_, err = BuildQuery(cmd, jsonStyle)
	if err == nil || !strings.Contains(err.Error(), "invalid filter JSON:") || strings.Contains(err.Error(), "JSON object") {
		t.Fatalf("BuildQuery() error = %v, want QueryObjectJSON wording", err)
	}
}
