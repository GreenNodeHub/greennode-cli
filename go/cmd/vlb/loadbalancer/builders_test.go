package loadbalancer

import (
	"reflect"
	"testing"
)

// Only the filters the live API actually honours are sent. Verified against
// HCM-3: `name` narrows the result set and `zoneId` matches exactly, while
// ids/subnetIds/vpcIds/status and any pagination parameter behave exactly like
// a parameter that does not exist.
func TestBuildListQuery(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		zoneID string
		want   map[string]string
	}{
		{
			name: "no filters sends no parameters at all",
			want: map[string]string{},
		},
		{
			name:   "name only",
			filter: "web",
			want:   map[string]string{"name": "web"},
		},
		{
			name:   "zone only",
			zoneID: "HCM03-1A",
			want:   map[string]string{"zoneId": "HCM03-1A"},
		},
		{
			name:   "both",
			filter: "web",
			zoneID: "HCM03-1A",
			want:   map[string]string{"name": "web", "zoneId": "HCM03-1A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildListQuery(tt.filter, tt.zoneID)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildListQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestListDoesNotOfferUnsupportedFilters pins the decision above: these flags
// were removed because the backend ignores them, and re-adding one without the
// backend implementing it would silently mislead users.
func TestListDoesNotOfferUnsupportedFilters(t *testing.T) {
	for _, name := range []string{"page", "page-size", "ids", "subnet-ids", "vpc-ids", "status"} {
		if listCmd.Flags().Lookup(name) != nil {
			t.Errorf("`loadbalancer list` declares --%s, but the API ignores it; "+
				"only re-add it once the backend honours it", name)
		}
	}
	for _, name := range []string{"name", "zone-id"} {
		if listCmd.Flags().Lookup(name) == nil {
			t.Errorf("`loadbalancer list` should offer --%s (the API honours it)", name)
		}
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Run("required fields only", func(t *testing.T) {
		got, err := buildCreateBody("lb1", "pkg1", "Internet", "sub1", "Layer 7", "", false, nil, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]interface{}{
			"name": "lb1", "packageId": "pkg1", "scheme": "Internet",
			"subnetId": "sub1", "type": "Layer 7",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v, want %v", got, want)
		}
	})

	t.Run("optional keys are absent when unset", func(t *testing.T) {
		got, _ := buildCreateBody("lb1", "pkg1", "Internal", "sub1", "Layer 4", "", false, nil, "", "")
		for _, key := range []string{"zoneId", "autoScalable", "tags", "listener", "pool"} {
			if _, ok := got[key]; ok {
				t.Errorf("key %q should be absent when its flag is unset", key)
			}
		}
	})

	t.Run("nested listener shorthand coerces int fields", func(t *testing.T) {
		got, err := buildCreateBody("lb1", "pkg1", "Internet", "sub1", "Layer 7", "zone1", true,
			[]string{"key=env,value=prod"},
			"listenerName=web,listenerProtocol=HTTP,listenerProtocolPort=80",
			"poolName=p,poolProtocol=HTTP,algorithm=ROUND_ROBIN,stickiness=true")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		listener := got["listener"].(map[string]interface{})
		if listener["listenerProtocolPort"] != 80 {
			t.Errorf("listenerProtocolPort = %#v, want int 80", listener["listenerProtocolPort"])
		}
		pool := got["pool"].(map[string]interface{})
		if pool["stickiness"] != true {
			t.Errorf("stickiness = %#v, want bool true", pool["stickiness"])
		}
		if got["zoneId"] != "zone1" || got["autoScalable"] != true {
			t.Errorf("zoneId/autoScalable not carried through: %v", got)
		}
	})

	t.Run("malformed shorthand is rejected", func(t *testing.T) {
		if _, err := buildCreateBody("lb1", "pkg1", "Internet", "sub1", "Layer 7", "", false, nil,
			"listenerProtocolPort=notanumber", ""); err == nil {
			t.Error("expected an error for a non-integer port")
		}
		if _, err := buildCreateBody("lb1", "pkg1", "Internet", "sub1", "Layer 7", "", false, nil,
			"missing-equals", ""); err == nil {
			t.Error("expected an error for shorthand without '='")
		}
	})
}

func TestBuildScalingBody(t *testing.T) {
	t.Run("minSize is omitted unless explicitly set", func(t *testing.T) {
		got, err := buildScalingBody(0, 5, false, "sub-1,sub-2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		scaling := got["scaling"].(map[string]interface{})
		if _, ok := scaling["minSize"]; ok {
			t.Error("minSize should be absent when --min-size was not set")
		}
		if scaling["maxSize"] != 5 {
			t.Errorf("maxSize = %v, want 5", scaling["maxSize"])
		}
		networking := got["networking"].(map[string]interface{})
		if !reflect.DeepEqual(networking["subnets"], []string{"sub-1", "sub-2"}) {
			t.Errorf("subnets = %v", networking["subnets"])
		}
	})

	t.Run("minSize is sent when set, even as zero", func(t *testing.T) {
		got, _ := buildScalingBody(0, 3, true, "sub-1")
		scaling := got["scaling"].(map[string]interface{})
		if scaling["minSize"] != 0 {
			t.Errorf("minSize = %v, want 0", scaling["minSize"])
		}
	})

	t.Run("rejects an empty subnet list and a non-positive maxSize", func(t *testing.T) {
		if _, err := buildScalingBody(1, 5, true, " , "); err == nil {
			t.Error("expected an error for an empty --subnets")
		}
		if _, err := buildScalingBody(1, 0, true, "sub-1"); err == nil {
			t.Error("expected an error for --max-size 0")
		}
	})
}

func TestBuildCloneBody(t *testing.T) {
	got, err := buildCloneBody("clone-1", "pkg-2", []string{
		"listenerId=li-1,defaultCertificateId=ca-1,sniCertificateIds=ca-2|ca-3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	certs := got["certificates"].([]map[string]interface{})
	if len(certs) != 1 {
		t.Fatalf("certificates length = %d, want 1", len(certs))
	}
	if !reflect.DeepEqual(certs[0]["sniCertificateIds"], []string{"ca-2", "ca-3"}) {
		t.Errorf("sniCertificateIds = %#v, want a []string split on '|'", certs[0]["sniCertificateIds"])
	}

	bare, _ := buildCloneBody("clone-1", "pkg-2", nil)
	if _, ok := bare["certificates"]; ok {
		t.Error("certificates should be absent when no --certificate was passed")
	}
}

// TagRequest is {key, value}. The natural guess --tag env=prod parses to
// {"env":"prod"}, which the API cannot use, so it must fail with a hint.
func TestBuildCreateBodyChecksTagShape(t *testing.T) {
	for _, tag := range []string{"env=prod", "value=prod", "key=env,value=prod,extra=1"} {
		if _, err := buildCreateBody("lb1", "pkg1", "Internet", "sub1", "Layer 7", "", false,
			[]string{tag}, "", ""); err == nil {
			t.Errorf("--tag %s: expected an error", tag)
		}
	}
	for _, tag := range []string{"key=env,value=prod", "key=env", `{"key":"env","value":"prod"}`} {
		if _, err := buildCreateBody("lb1", "pkg1", "Internet", "sub1", "Layer 7", "", false,
			[]string{tag}, "", ""); err != nil {
			t.Errorf("--tag %s: unexpected error: %v", tag, err)
		}
	}
}
