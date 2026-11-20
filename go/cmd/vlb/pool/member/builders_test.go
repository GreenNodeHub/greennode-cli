package member

import (
	"github.com/spf13/pflag"
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
		got, err := buildUpdateBody(currentMembers(), []string{"ipAddress=10.0.0.9,port=8080,monitorPort=8080"}, true, false)
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
		got, err := buildUpdateBody(currentMembers(), []string{"ipAddress=10.0.0.9,port=8080,monitorPort=8080"}, true, true)
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
		if _, err := buildUpdateBody(currentMembers(), nil, false, false); err == nil {
			t.Error("expected an error rather than silently emptying the pool")
		}
	})

	t.Run("an explicit empty --member empties the pool", func(t *testing.T) {
		got, err := buildUpdateBody(currentMembers(), nil, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if members := got["members"].([]map[string]interface{}); len(members) != 0 {
			t.Errorf("members = %v, want an empty list", members)
		}
	})

	t.Run("malformed shorthand is rejected", func(t *testing.T) {
		if _, err := buildUpdateBody(nil, []string{"ipAddress=10.0.0.9,port=eighty"}, true, false); err == nil {
			t.Error("expected an error for a non-integer port")
		}
	})

	t.Run("a new member without monitorPort is rejected", func(t *testing.T) {
		_, err := buildUpdateBody(nil, []string{"ipAddress=10.0.0.9,port=8080"}, true, false)
		if err == nil || !strings.Contains(err.Error(), "monitorPort is required") {
			t.Errorf("error = %v, want the monitorPort requirement", err)
		}
	})

	t.Run("--keep-existing: a --member matching ipAddress:port replaces the current one", func(t *testing.T) {
		got, err := buildUpdateBody(currentMembers(),
			[]string{"ipAddress=10.0.0.4,port=80,monitorPort=80,weight=5"}, true, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		members := got["members"].([]map[string]interface{})
		if len(members) != 2 {
			t.Fatalf("members length = %d, want 2 (no duplicate 10.0.0.4:80)", len(members))
		}
		// Replaced in place, keeping the order of the current list.
		if members[0]["ipAddress"] != "10.0.0.4" || members[0]["weight"] != 5 {
			t.Errorf("members[0] = %#v, want 10.0.0.4 with weight 5", members[0])
		}
		if members[1]["ipAddress"] != "10.0.0.5" {
			t.Errorf("members[1] = %#v, want the untouched 10.0.0.5", members[1])
		}
	})

	t.Run("repeating the same ipAddress:port in --member keeps only the last", func(t *testing.T) {
		got, err := buildUpdateBody(nil, []string{
			"ipAddress=10.0.0.9,port=80,monitorPort=80,weight=1",
			"ipAddress=10.0.0.9,port=80,monitorPort=80,weight=3",
		}, true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		members := got["members"].([]map[string]interface{})
		if len(members) != 1 || members[0]["weight"] != 3 {
			t.Errorf("members = %v, want one member with weight 3", members)
		}
	})

	t.Run("--keep-existing without --member is refused (it would be a no-op PUT)", func(t *testing.T) {
		_, err := buildUpdateBody(currentMembers(), nil, false, true)
		if err == nil || !strings.Contains(err.Error(), "nothing to update") {
			t.Errorf("error = %v, want a nothing-to-update error", err)
		}
	})

	t.Run("--keep-existing does not re-check members carried over from the API", func(t *testing.T) {
		// currentMembers() have no monitorPort; they came from the service, so
		// they must not be rejected on the way back.
		if _, err := buildUpdateBody(currentMembers(), []string{"ipAddress=10.0.0.9,port=8080,monitorPort=8080"}, true, true); err != nil {
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

// TestEmptyMemberFlagReachesTheBuilderAsPflagDeliversIt drives the real flag set
// rather than hand-writing the slice. pflag collapses an empty stringArray
// occurrence, so `--member ”` arrives as an EMPTY slice with Changed=true —
// never as []string{""}. An earlier test asserted the hand-written []string{""}
// and so passed while the command itself refused the only documented way to
// empty a pool, answering with the very error that prescribes it.
func TestEmptyMemberFlagReachesTheBuilderAsPflagDeliversIt(t *testing.T) {
	for _, argv := range [][]string{
		{"--member", ""},
		{"--member="},
	} {
		f := pflag.NewFlagSet("member-update", pflag.ContinueOnError)
		f.StringArray("member", nil, "")
		if err := f.Parse(argv); err != nil {
			t.Fatalf("%v: parse: %v", argv, err)
		}

		flags, _ := f.GetStringArray("member")
		if len(flags) != 0 {
			t.Fatalf("%v: pflag delivered %#v; this test's premise no longer holds", argv, flags)
		}
		if !f.Changed("member") {
			t.Fatalf("%v: Changed is false, so the builder cannot tell empty from absent", argv)
		}

		got, err := buildUpdateBody(currentMembers(), flags, f.Changed("member"), false)
		if err != nil {
			t.Fatalf("%v: unexpected error: %v", argv, err)
		}
		members, ok := got["members"].([]map[string]interface{})
		if !ok || len(members) != 0 {
			t.Errorf("%v: members = %#v, want an empty list", argv, got["members"])
		}
	}
}

// The guard must still fire when the flag was never passed, so a bare
// `pool member update` cannot silently wipe the pool.
func TestAbsentMemberFlagIsStillRefused(t *testing.T) {
	f := pflag.NewFlagSet("member-update", pflag.ContinueOnError)
	f.StringArray("member", nil, "")
	if err := f.Parse(nil); err != nil {
		t.Fatal(err)
	}
	flags, _ := f.GetStringArray("member")

	_, err := buildUpdateBody(currentMembers(), flags, f.Changed("member"), false)
	if err == nil || !strings.Contains(err.Error(), "provide at least one --member") {
		t.Errorf("error = %v, want the refusal", err)
	}
}

// backup is @NotNull on MemberRequest; omitting it returned a 400 naming a field
// the user never set.
func TestMemberUpdateFillsInBackup(t *testing.T) {
	got, err := buildUpdateBody(nil, []string{"ipAddress=10.0.0.9,port=8080,monitorPort=8080"}, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	members := got["members"].([]map[string]interface{})
	if v, ok := members[0]["backup"]; !ok || v != false {
		t.Errorf("backup = %#v (present=%v), want false to be filled in", v, ok)
	}
}

// member update shares pool create's member checks, so a member without a
// port fails here instead of reaching the API's "Invalid port number".
func TestMemberUpdateRequiresIPAddressAndPort(t *testing.T) {
	for _, member := range []string{"monitorPort=80,weight=1", "ipAddress=10.0.0.4,monitorPort=80"} {
		if _, err := buildUpdateBody(nil, []string{member}, true, false); err == nil {
			t.Errorf("--member %s: expected an error", member)
		}
	}
}
