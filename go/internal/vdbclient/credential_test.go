package vdbclient

import "testing"

// TestPasswordRulesDifferPerProduct pins the two length ranges apart. They overlap only
// between 16 and 32, so a password that suits one product can be invalid for the other —
// which is the trap when $GRN_VDB_MASTER_PASSWORD is reused.
func TestPasswordRulesDifferPerProduct(t *testing.T) {
	const (
		eight     = "Abcd1234"                                                           // valid for relational, too short for Redis
		twenty    = "Abcd1234Abcd1234Abcd"                                               // valid for both
		fortyFour = "Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd"                       // too long for relational, fine for Redis
		huge      = "Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234" + //
			"Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234x" // 129
	)

	for name, tc := range map[string]struct {
		password string
		db, rd   bool // want accepted by relational / by Redis
	}{
		"7 characters":   {"Abcd123", false, false},
		"8 characters":   {eight, true, false},
		"20 characters":  {twenty, true, true},
		"44 characters":  {fortyFour, false, true},
		"129 characters": {huge, false, false},
	} {
		if err := ValidateDBPassword(tc.password, "password"); (err == nil) != tc.db {
			t.Errorf("%s: ValidateDBPassword = %v, want accepted=%t", name, err, tc.db)
		}
		if err := ValidateRedisPassword(tc.password, "redis-password"); (err == nil) != tc.rd {
			t.Errorf("%s: ValidateRedisPassword = %v, want accepted=%t", name, err, tc.rd)
		}
	}
}

// TestPasswordCharacterSet: letters, digits and $ ^ _ < > are allowed and everything else
// is not. The rejected list is what a live probe actually saw refused.
func TestPasswordCharacterSet(t *testing.T) {
	const base = "Abcd1234Abcd1234" // 16, valid for both products

	for _, allowed := range []string{"$", "^", "_", "<", ">"} {
		if err := ValidateRedisPassword(base+allowed, "redis-password"); err != nil {
			t.Errorf("%q rejected but allowed by the product rule: %v", allowed, err)
		}
	}

	for _, rejected := range []string{"-", ".", "@", "#", "%", "*", "+", "=", "!", " ", "'", "\\", "ă"} {
		err := ValidateRedisPassword(base+rejected, "redis-password")
		if err == nil {
			t.Errorf("%q accepted, want a rejection", rejected)
			continue
		}
		// The message must name the flag and the offending character, so the user can see
		// which of their inputs to fix without echoing the whole secret.
		for _, want := range []string{"--redis-password", rejected} {
			if !contains(err.Error(), want) {
				t.Errorf("error for %q = %q, must mention %q", rejected, err, want)
			}
		}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
