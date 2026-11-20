package vlbclient

import "fmt"

// UnwrapData returns the payload of a {"data": {...}} envelope, or the response
// itself when it is already the bare object (see the five envelope shapes in
// cmd/vlb/CLAUDE.md). A response that is not an object gives an empty map.
func UnwrapData(response interface{}) map[string]interface{} {
	m, ok := response.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	if data, ok := m["data"].(map[string]interface{}); ok {
		return data
	}
	return m
}

// MemberIntFields / MemberBoolFields drive shorthand type coercion for the
// --member struct flag of `pool create` and `pool member update`; they mirror
// MemberRequest.
var (
	MemberIntFields  = []string{"port", "weight", "monitorPort"}
	MemberBoolFields = []string{"backup"}
)

// ValidateMembers checks parsed --member entries against what the API really
// requires, and fills in backup=false when it is missing.
//
//   - ipAddress and port are required by MemberRequest.
//   - monitorPort is optional in the spec but mandatory in practice: without it
//     the API answers "members[N].monitorPort: Invalid port number", which is
//     confusing when the user never set one.
//   - backup is a boxed @NotNull Boolean, so omitting it fails with "must not be
//     null". Its default is clear, so it is filled in rather than demanded.
//
// Members read back from the API (pool member update --keep-existing) must not
// be passed here; they already carry the service's own values.
func ValidateMembers(members []map[string]interface{}) error {
	for i, m := range members {
		if _, ok := m["ipAddress"]; !ok {
			return fmt.Errorf("--member[%d]: ipAddress is required", i)
		}
		if _, ok := m["port"]; !ok {
			return fmt.Errorf("--member[%d]: port is required", i)
		}
		if _, ok := m["monitorPort"]; !ok {
			return fmt.Errorf("--member[%d]: monitorPort is required by the API "+
				"(set it to the same value as port for a plain health check)", i)
		}
		if _, ok := m["backup"]; !ok {
			m["backup"] = false
		}
	}
	return nil
}
