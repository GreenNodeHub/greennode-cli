package member

import (
	"reflect"
	"strings"
	"testing"
)

// currentMembers is a trimmed PoolMemberDto list as returned by GET.
func currentMembers() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"uuid": "m-1", "address": "10.0.0.4", "protocolPort": float64(80),
			"weight": float64(1), "name": "web-1", "backup": false,
			"displayStatus": "ACTIVE", "poolId": "pool-1",
		},
		map[string]interface{}{
			"uuid": "m-2", "address": "10.0.0.5", "protocolPort": float64(80),
			"weight": float64(2), "name": "web-2", "backup": true,
		},
	}
}

func TestBuildUpdateBody(t *testing.T) {
	t.Run("without --keep-existing the list is replaced wholesale", func(t *testing.T) {
		got, err := buildUpdateBody(currentMembers(), []string{"ipAddress=10.0.0.9,port=8080,monitorPort=8080"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		members := got["members"].([]map[string]interface{})
		if len(members) != 1 {
			t.Fatalf("members length = %d, want only the 1 new member", len(members))
		}
		if members[0]["ipAddress"] != "10.0.0.9" || members[0]["port"] != 8080 {
			t.Errorf("member = %#v", members[0])
		}
	})

	t.Run("--keep-existing converts current members to request shape and appends", func(t *testing.T) {
		got, err := buildUpdateBody(currentMembers(), []string{"ipAddress=10.0.0.9,port=8080,monitorPort=8080"}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		members := got["members"].([]map[string]interface{})
		if len(members) != 3 {
			t.Fatalf("members length = %d, want 2 existing + 1 new", len(members))
		}
		// PoolMemberDto.address/protocolPort become MemberRequest.ipAddress/port.
		if members[0]["ipAddress"] != "10.0.0.4" || members[0]["port"] != float64(80) {
			t.Errorf("existing member not converted: %#v", members[0])
		}
		// Read-only DTO fields must not be echoed back.
		for _, readOnly := range []string{"uuid", "displayStatus", "poolId"} {
			if _, ok := members[0][readOnly]; ok {
				t.Errorf("read-only field %q must not be sent back", readOnly)
			}
		}
		if members[2]["ipAddress"] != "10.0.0.9" {
			t.Errorf("new member should be appended last, got %#v", members[2])
		}
	})

	t.Run("passing no --member at all is refused", func(t *testing.T) {
		if _, err := buildUpdateBody(currentMembers(), nil, false); err == nil {
			t.Error("expected an error rather than silently emptying the pool")
		}
	})

	t.Run("an explicit empty --member empties the pool", func(t *testing.T) {
		got, err := buildUpdateBody(currentMembers(), []string{""}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if members := got["members"].([]map[string]interface{}); len(members) != 0 {
			t.Errorf("members = %v, want an empty list", members)
		}
	})

	t.Run("malformed shorthand is rejected", func(t *testing.T) {
		if _, err := buildUpdateBody(nil, []string{"ipAddress=10.0.0.9,port=eighty"}, false); err == nil {
			t.Error("expected an error for a non-integer port")
		}
	})

	t.Run("a new member without monitorPort is rejected", func(t *testing.T) {
		_, err := buildUpdateBody(nil, []string{"ipAddress=10.0.0.9,port=8080"}, false)
		if err == nil || !strings.Contains(err.Error(), "monitorPort is required") {
			t.Errorf("error = %v, want the monitorPort requirement", err)
		}
	})

	t.Run("--keep-existing does not re-check members carried over from the API", func(t *testing.T) {
		// currentMembers() have no monitorPort; they came from the service, so
		// they must not be rejected on the way back.
		if _, err := buildUpdateBody(currentMembers(), []string{"ipAddress=10.0.0.9,port=8080,monitorPort=8080"}, true); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestMemberList(t *testing.T) {
	wrapped := map[string]interface{}{"data": currentMembers()}
	if got := memberList(wrapped); len(got) != 2 {
		t.Errorf("memberList(envelope) length = %d, want 2", len(got))
	}
	if got := memberList(currentMembers()); len(got) != 2 {
		t.Errorf("memberList(bare array) length = %d, want 2", len(got))
	}
	if got := memberList(map[string]interface{}{}); got != nil {
		t.Errorf("memberList(empty) = %v, want nil", got)
	}
}

func TestCurrentMemberFieldsMapping(t *testing.T) {
	want := map[string]string{
		"address": "ipAddress", "protocolPort": "port", "weight": "weight",
		"name": "name", "monitorPort": "monitorPort", "backup": "backup",
	}
	if !reflect.DeepEqual(currentMemberFields, want) {
		t.Errorf("currentMemberFields = %v, want %v", currentMemberFields, want)
	}
}
