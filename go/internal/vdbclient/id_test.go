package vdbclient

import (
	"strings"
	"testing"
)

func TestRequireIDWithPrefix(t *testing.T) {
	const hint = "use 'grn vdb relational instance' instead"

	if err := RequireIDWithPrefix("pg-2e6f2253-9032-466f", "cluster-id", "pg-", hint); err != nil {
		t.Errorf("valid ID rejected: %v", err)
	}

	// The wrong product's ID is the case this exists for: several relational
	// endpoints serve both, so passing a db- ID to a cluster command would act on
	// a Relational Database instance.
	err := RequireIDWithPrefix("db-66a37ca3-e688", "cluster-id", "pg-", hint)
	if err == nil {
		t.Fatal("db- ID accepted for a pg- flag")
	}
	if !strings.Contains(err.Error(), hint) {
		t.Errorf("error %q does not carry the hint", err)
	}
	if !strings.Contains(err.Error(), "cluster-id") {
		t.Errorf("error %q does not name the flag", err)
	}

	// Invalid characters are still rejected by the underlying validator, before
	// any prefix check.
	if err := RequireIDWithPrefix("pg-1; rm -rf /", "cluster-id", "pg-", hint); err == nil {
		t.Error("ID with unsafe characters accepted")
	}
	if err := RequireIDWithPrefix("", "cluster-id", "pg-", hint); err == nil {
		t.Error("empty ID accepted")
	}

	// Without a hint the message must still be complete.
	if err := RequireIDWithPrefix("db-1", "cluster-id", "pg-", ""); err == nil ||
		!strings.Contains(err.Error(), "pg-") {
		t.Errorf("error without a hint = %v", err)
	}
}
