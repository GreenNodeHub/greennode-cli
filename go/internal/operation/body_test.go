package operation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newBodyTestCommand(t *testing.T, raw string, set bool) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("body", "", "Request body")
	if set {
		if err := cmd.Flags().Set("body", raw); err != nil {
			t.Fatal(err)
		}
	}
	return cmd
}

func TestParseBodyNilContractReturnsNil(t *testing.T) {
	body, err := ParseBody(&cobra.Command{Use: "test"}, nil)
	if err != nil || body != nil {
		t.Fatalf("ParseBody(nil contract) = (%#v, %v), want (nil, nil)", body, err)
	}
}

func TestParseBodyRequiresNonEmptyFlag(t *testing.T) {
	cmd := newBodyTestCommand(t, "", false)
	contract := &BodyContract{Kind: ObjectBody}
	if _, err := ParseBody(cmd, contract); err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("ParseBody() error = %v, want empty-body rejection", err)
	}
}

func TestParseBodyOptionalObjectAllowsAbsentFlag(t *testing.T) {
	cmd := newBodyTestCommand(t, "", false)
	contract := &BodyContract{Kind: OptionalObjectBody}
	body, err := ParseBody(cmd, contract)
	if err != nil || body != nil {
		t.Fatalf("ParseBody() = (%#v, %v), want (nil, nil) for an absent optional body", body, err)
	}
}

func TestParseBodyUsesJSONNumberPrecision(t *testing.T) {

	cmd := newBodyTestCommand(t, `{"id": 9007199254740993}`, true)
	contract := &BodyContract{Kind: ObjectBody}
	body, err := ParseBody(cmd, contract)
	if err != nil {
		t.Fatalf("ParseBody() error = %v", err)
	}
	object, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("body = %#v, want a JSON object", body)
	}
	number, ok := object["id"].(json.Number)
	if !ok {
		t.Fatalf("body[id] = %#v (%T), want json.Number", object["id"], object["id"])
	}
	if number.String() != "9007199254740993" {
		t.Fatalf("body[id] = %s, want the exact original digits", number.String())
	}
}

func TestParseBodyRejectsMultipleJSONValues(t *testing.T) {
	cmd := newBodyTestCommand(t, `{}{}`, true)
	contract := &BodyContract{Kind: ObjectBody}
	if _, err := ParseBody(cmd, contract); err == nil || !strings.Contains(err.Error(), "multiple JSON values are not allowed") {
		t.Fatalf("ParseBody() error = %v, want multi-value rejection", err)
	}
}

func TestParseBodyRejectsTrailingGarbageAfterValidJSON(t *testing.T) {
	cmd := newBodyTestCommand(t, `{} not json`, true)
	contract := &BodyContract{Kind: ObjectBody}
	if _, err := ParseBody(cmd, contract); err == nil {
		t.Fatal("ParseBody() error = nil, want trailing-content rejection")
	}
}

func TestParseBodyRequiresObjectShape(t *testing.T) {
	cmd := newBodyTestCommand(t, `[1,2,3]`, true)
	contract := &BodyContract{Kind: ObjectBody}
	if _, err := ParseBody(cmd, contract); err == nil || !strings.Contains(err.Error(), "must be a JSON object") {
		t.Fatalf("ParseBody() error = %v, want object-shape rejection", err)
	}
}

func TestParseBodyRequiresArrayShape(t *testing.T) {
	cmd := newBodyTestCommand(t, `{}`, true)
	contract := &BodyContract{Kind: ArrayBody}
	if _, err := ParseBody(cmd, contract); err == nil || !strings.Contains(err.Error(), "must be a JSON array") {
		t.Fatalf("ParseBody() error = %v, want array-shape rejection", err)
	}
}

func TestParseBodyArrayPermitsNonObjectItemsByDefault(t *testing.T) {
	// vstorage's requiredArrayBody: no StrictArrayItems, so array elements
	// may be any JSON value.
	cmd := newBodyTestCommand(t, `[1, "two", true]`, true)
	contract := &BodyContract{Kind: ArrayBody}
	if _, err := ParseBody(cmd, contract); err != nil {
		t.Fatalf("ParseBody() error = %v, want permissive array items accepted", err)
	}
}

func TestParseBodyStrictArrayItemsRequireObjectsAndFields(t *testing.T) {
	// vdb's arrayBody: every element must be an object, and required item
	// fields are enforced per element.
	contract := &BodyContract{Kind: ArrayBody, StrictArrayItems: true, Name: "DeleteBackupRequest", RequiredItemFields: []string{"backupId"}}

	cmd := newBodyTestCommand(t, `[1]`, true)
	if _, err := ParseBody(cmd, contract); err == nil || !strings.Contains(err.Error(), "must be a JSON object") {
		t.Fatalf("ParseBody() error = %v, want non-object item rejection", err)
	}

	cmd = newBodyTestCommand(t, `[{}]`, true)
	if _, err := ParseBody(cmd, contract); err == nil || !strings.Contains(err.Error(), `missing required field "backupId"`) {
		t.Fatalf("ParseBody() error = %v, want missing-required-item-field rejection", err)
	}

	cmd = newBodyTestCommand(t, `[{"backupId":"b-1"}]`, true)
	if _, err := ParseBody(cmd, contract); err != nil {
		t.Fatalf("ParseBody() error = %v, want a satisfied item accepted", err)
	}
}

func TestParseBodyChecksTopLevelRequiredFields(t *testing.T) {
	contract := &BodyContract{Kind: ObjectBody, Name: "CreateThingRequest", RequiredFields: []string{"name"}}
	cmd := newBodyTestCommand(t, `{"other":"x"}`, true)
	if _, err := ParseBody(cmd, contract); err == nil || !strings.Contains(err.Error(), `missing required field "name" for CreateThingRequest`) {
		t.Fatalf("ParseBody() error = %v, want required-field rejection naming the schema", err)
	}
}

func TestParseBodyStyleSelectsDecodeErrorWording(t *testing.T) {
	cases := []struct {
		style BodyStyle
		want  string
	}{
		{BodyStyleJSONBody, "invalid JSON body:"},
		{BodyStyleJSONObject, "invalid body JSON object:"},
		{BodyStyleJSON, "invalid body JSON:"},
	}
	for _, tt := range cases {
		cmd := newBodyTestCommand(t, `not json`, true)
		contract := &BodyContract{Kind: ObjectBody, Style: tt.style}
		_, err := ParseBody(cmd, contract)
		if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
			t.Errorf("style %d: ParseBody() error = %v, want prefix %q", tt.style, err, tt.want)
		}
	}
}

func TestOptionalJSONBodyRejectsInvalidJSON(t *testing.T) {
	for _, raw := range []string{`{`, `{}[]`, `[] garbage`} {
		if _, err := ParseBody(newBodyTestCommand(t, raw, true), &BodyContract{Kind: OptionalJSONBody}); err == nil {
			t.Errorf("accepted invalid JSON %q", raw)
		}
	}
}
