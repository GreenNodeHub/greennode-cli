package operation

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newPathTestCommand(params []PathParam) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	for _, p := range params {
		cmd.Flags().String(p.Flag, "", p.Usage)
	}
	return cmd
}

func TestBuildPathSubstitutesValidatedPlaceholder(t *testing.T) {
	params := []PathParam{{Placeholder: "id", Flag: "id", Usage: "ID"}}
	cmd := newPathTestCommand(params)
	if err := cmd.Flags().Set("id", "abc-123"); err != nil {
		t.Fatal(err)
	}
	path, err := BuildPath(cmd, "TestSvc", "/v1/things/{id}", params)
	if err != nil {
		t.Fatalf("BuildPath() error = %v", err)
	}
	if path != "/v1/things/abc-123" {
		t.Fatalf("path = %q, want /v1/things/abc-123", path)
	}
}

func TestBuildPathEscapesSpecialCharactersFromCustomValidator(t *testing.T) {

	params := []PathParam{{
		Placeholder: "object",
		Flag:        "object",
		Usage:       "Object key",
		Validate:    func(string, string) error { return nil },
	}}
	cmd := newPathTestCommand(params)
	if err := cmd.Flags().Set("object", "a b/c"); err != nil {
		t.Fatal(err)
	}
	path, err := BuildPath(cmd, "TestSvc", "/v1/objects/{object}", params)
	if err != nil {
		t.Fatalf("BuildPath() error = %v", err)
	}
	if path != "/v1/objects/a%20b%2Fc" {
		t.Fatalf("path = %q, want percent-escaped segment", path)
	}
}

func TestBuildPathDefaultsToValidateID(t *testing.T) {
	params := []PathParam{{Placeholder: "id", Flag: "id", Usage: "ID"}}
	cmd := newPathTestCommand(params)
	if err := cmd.Flags().Set("id", "not a valid id"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildPath(cmd, "TestSvc", "/v1/things/{id}", params); err == nil {
		t.Fatal("BuildPath() error = nil, want validator.ValidateID rejection")
	}
}

func TestBuildPathCustomValidatorRejectsValue(t *testing.T) {
	params := []PathParam{{
		Placeholder: "domain",
		Flag:        "domain",
		Usage:       "Domain",
		Validate: func(value, flag string) error {
			if !strings.Contains(value, ".") {
				return &pathValidationError{flag: flag}
			}
			return nil
		},
	}}
	cmd := newPathTestCommand(params)
	if err := cmd.Flags().Set("domain", "not-a-domain"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildPath(cmd, "TestSvc", "/v1/domains/{domain}", params); err == nil {
		t.Fatal("BuildPath() error = nil, want custom validator rejection")
	}
}

type pathValidationError struct{ flag string }

func (e *pathValidationError) Error() string { return "invalid " + e.flag }

func TestBuildPathReportsUnresolvedPlaceholder(t *testing.T) {

	cmd := newPathTestCommand(nil)
	_, err := BuildPath(cmd, "TestSvc", "/v1/things/{id}", nil)
	if err == nil {
		t.Fatal("BuildPath() error = nil, want unresolved-placeholder error")
	}
	want := "internal TestSvc contract error: unresolved path parameter in /v1/things/{id}"
	if err.Error() != want {
		t.Fatalf("BuildPath() error = %q, want %q", err.Error(), want)
	}
}

func TestBuildPathWithNoParametersReturnsTemplateUnchanged(t *testing.T) {
	cmd := newPathTestCommand(nil)
	path, err := BuildPath(cmd, "TestSvc", "/v1/regions", nil)
	if err != nil || path != "/v1/regions" {
		t.Fatalf("BuildPath() = (%q, %v), want (/v1/regions, nil)", path, err)
	}
}
