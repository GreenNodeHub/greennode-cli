package vdbclient

import (
	"fmt"
	"strings"
)

// Master-password rules for vDB, checked before the request goes out so a rejection
// names the user's own flag instead of arriving as a 400 after the secret has crossed
// the network.
//
// Both rules are product rules, stated by the API owner on 2026-08-13:
//
//   - Characters: letters, digits and $ ^ _ < > only, in both products. A live probe on
//     the MemoryStore API agrees — each of $ ^ _ < > was accepted on its own, while
//   - . @ # % * + = ! came back as "Redis password contains invalid character".
//   - One undocumented extra, found by that probe: MemoryStore also rejects a password
//     containing BOTH < and >, in either order, however far apart — a tag-like sequence
//     its sanitiser refuses. Either bracket alone is fine. Enforced for MemoryStore only,
//     since only that API was probed for it.
//   - Length: 8 to 32 for Relational Database, 16 to 128 for MemoryStore. The
//     MemoryStore bound was also confirmed live at both edges (15 and 129 rejected, 16
//     and 128 accepted), and its API states it as "The field Password must be at least
//     16 and max 128 (range_value)".
//
// The two ranges overlap only between 16 and 32, which is worth knowing when reusing one
// $GRN_VDB_MASTER_PASSWORD across products.
//
// PostgreSQL Cluster is covered by neither: it is a separate API surface that has
// already proven to validate differently from relational (it accepts an empty
// configGroupId, which relational rejects), so its password rules stay unverified and
// are left to the API, whose message the error layer surfaces in full.
const (
	minDBPasswordLen    = 8
	maxDBPasswordLen    = 32
	minRedisPasswordLen = 16
	maxRedisPasswordLen = 128
	passwordExtraChars  = "$^_<>"
)

// ValidateDBPassword checks a Relational Database master password.
func ValidateDBPassword(password, flagName string) error {
	return validatePassword(password, flagName, minDBPasswordLen, maxDBPasswordLen, false)
}

// ValidateRedisPassword checks a MemoryStore master password, whose length bound differs
// from the relational one.
func ValidateRedisPassword(password, flagName string) error {
	return validatePassword(password, flagName, minRedisPasswordLen, maxRedisPasswordLen, true)
}

// rejectBracketPair carries the MemoryStore-only rule; see the note above.
func validatePassword(password, flagName string, min, max int, rejectBracketPair bool) error {
	if length := len([]rune(password)); length < min || length > max {
		return fmt.Errorf("--%s must be between %d and %d characters, got %d", flagName, min, max, length)
	}
	if rejectBracketPair && strings.ContainsRune(password, '<') && strings.ContainsRune(password, '>') {
		return fmt.Errorf("--%s cannot contain both < and >: the API rejects the pair, though either alone is allowed", flagName)
	}
	for _, r := range password {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune(passwordExtraChars, r):
		default:
			return fmt.Errorf("--%s contains %q: only letters, digits and %s are allowed",
				flagName, r, passwordExtraChars)
		}
	}
	return nil
}
