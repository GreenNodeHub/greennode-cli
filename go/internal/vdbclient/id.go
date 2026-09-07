package vdbclient

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/validator"
)

// RequireIDWithPrefix validates an ID and additionally enforces the product
// prefix vDB puts on it ("db-" for a Relational Database instance, "pg-" for a
// PostgreSQL Cluster).
//
// The prefix check is a safety measure, not cosmetics: several relational
// endpoints — get-by-id, histories, secrules, reboot, delete — serve BOTH
// products, so a command group that only means to touch one of them must reject
// the other's IDs itself. Without that, `grn vdb postgresql cluster delete
// --cluster-id db-...` would delete a Relational Database instance.
//
// hint is appended to the error to point at the command group that does handle
// the other prefix; pass "" to omit it.
func RequireIDWithPrefix(value, flagName, prefix, hint string) error {
	if err := validator.ValidateID(value, flagName); err != nil {
		return err
	}
	if !strings.HasPrefix(value, prefix) {
		msg := fmt.Sprintf("invalid %s: '%s' does not start with '%s'", flagName, value, prefix)
		if hint != "" {
			msg += ". " + hint
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
