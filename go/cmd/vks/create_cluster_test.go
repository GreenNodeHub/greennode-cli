package vks

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// subnetFlags builds the flag set resolveSubnetIDs reads, mirroring the flags
// declared in createClusterCmd.
func subnetFlags(subnetIDs, listSubnetIDs string) *pflag.FlagSet {
	f := pflag.NewFlagSet("create-cluster", pflag.ContinueOnError)
	f.String("subnet-ids", subnetIDs, "")
	f.String("list-subnet-ids", listSubnetIDs, "")
	return f
}

// createClusterFlags applies values to createClusterCmd's real flag set and
// restores every flag afterwards. Using the command's own flags means a renamed
// or missing flag fails the test instead of silently yielding a zero value.
func createClusterFlags(t *testing.T, values map[string]string) *pflag.FlagSet {
	t.Helper()
	f := createClusterCmd.Flags()
	t.Cleanup(func() {
		f.VisitAll(func(fl *pflag.Flag) {
			_ = fl.Value.Set(fl.DefValue)
			fl.Changed = false
		})
	})
	for name, v := range values {
		if f.Lookup(name) == nil {
			t.Fatalf("flag --%s does not exist on create-cluster", name)
		}
		if err := f.Set(name, v); err != nil {
			t.Fatalf("set --%s=%q: %v", name, v, err)
		}
	}
	return f
}

// TestBuildCreateClusterBody pins the wire payload: the API field is
// listSubnetIds (CreateClusterComboDto), the deprecated subnetId is never sent,
// and optional fields stay absent unless asked for.
func TestBuildCreateClusterBody(t *testing.T) {
	f := createClusterFlags(t, map[string]string{
		"name":         "my-cluster",
		"k8s-version":  "v1.30.10-vks.1746550800",
		"network-type": "CILIUM_OVERLAY",
		"vpc-id":       "net-aaa",
		"subnet-ids":   "sub-aaa,sub-bbb",
		"cidr":         "10.96.0.0/12",
	})

	body, err := buildCreateClusterBody(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, ok := body["listSubnetIds"].([]string)
	if !ok || !reflect.DeepEqual(ids, []string{"sub-aaa", "sub-bbb"}) {
		t.Errorf("listSubnetIds = %v (%T), want [sub-aaa sub-bbb]", body["listSubnetIds"], body["listSubnetIds"])
	}
	if _, present := body["subnetId"]; present {
		t.Error("subnetId must never be sent: it is deprecated on CreateClusterDto")
	}
	if body["azStrategy"] != "SINGLE" {
		t.Errorf("azStrategy = %v, want SINGLE (the flag default)", body["azStrategy"])
	}
	if body["vpcId"] != "net-aaa" || body["cidr"] != "10.96.0.0/12" || body["version"] != "v1.30.10-vks.1746550800" {
		t.Errorf("unexpected body: %v", body)
	}
	for _, absent := range []string{"description", "nodeNetmaskSize", "autoUpgradeConfig", "autoHealingConfig", "secondarySubnets"} {
		if _, present := body[absent]; present {
			t.Errorf("%s must be absent when its flag is unset, got %v", absent, body[absent])
		}
	}
}

func TestBuildCreateClusterBodyDeprecatedAlias(t *testing.T) {
	f := createClusterFlags(t, map[string]string{
		"name":            "my-cluster",
		"k8s-version":     "v1.30.10-vks.1746550800",
		"network-type":    "CILIUM_OVERLAY",
		"vpc-id":          "net-aaa",
		"list-subnet-ids": "sub-aaa",
	})

	body, err := buildCreateClusterBody(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids, _ := body["listSubnetIds"].([]string); !reflect.DeepEqual(ids, []string{"sub-aaa"}) {
		t.Errorf("listSubnetIds = %v, want [sub-aaa]", body["listSubnetIds"])
	}
}

func TestBuildCreateClusterBodyMissingSubnets(t *testing.T) {
	f := createClusterFlags(t, map[string]string{
		"name":         "my-cluster",
		"k8s-version":  "v1.30.10-vks.1746550800",
		"network-type": "CILIUM_OVERLAY",
		"vpc-id":       "net-aaa",
	})

	if _, err := buildCreateClusterBody(f); err == nil ||
		!strings.Contains(err.Error(), "--subnet-ids is required") {
		t.Errorf("err = %v, want --subnet-ids required error", err)
	}
}

// TestValidateAZStrategy covers the API rule that listSubnetIds is "a
// single-element list for SINGLE" — the default strategy, so a multi-subnet
// cluster needs MULTI spelled out.
func TestValidateAZStrategy(t *testing.T) {
	cases := []struct {
		name       string
		azStrategy string
		subnetIDs  []string
		wantErr    string
	}{
		{name: "single with one subnet", azStrategy: "SINGLE", subnetIDs: []string{"sub-aaa"}},
		{name: "multi with two subnets", azStrategy: "MULTI", subnetIDs: []string{"sub-aaa", "sub-bbb"}},
		{name: "multi with one subnet", azStrategy: "MULTI", subnetIDs: []string{"sub-aaa"}},
		{
			name:       "single with two subnets",
			azStrategy: "SINGLE",
			subnetIDs:  []string{"sub-aaa", "sub-bbb"},
			wantErr:    "--az-strategy SINGLE takes exactly one --subnet-ids value",
		},
		{
			name:       "unknown strategy",
			azStrategy: "multi",
			subnetIDs:  []string{"sub-aaa"},
			wantErr:    "--az-strategy must be SINGLE or MULTI",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateAZStrategy(tc.azStrategy, tc.subnetIDs)
			if tc.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("got %v, want no errors", errs)
				}
				return
			}
			if len(errs) != 1 || !strings.Contains(errs[0], tc.wantErr) {
				t.Errorf("got %v, want one error containing %q", errs, tc.wantErr)
			}
		})
	}
}

func TestValidateNodeNetmaskSize(t *testing.T) {
	cases := []struct {
		name    string
		set     bool
		size    int
		wantErr bool
	}{
		{name: "unset is fine", set: false, size: 0},
		{name: "24 ok", set: true, size: 24},
		{name: "26 ok", set: true, size: 26},
		{name: "23 rejected", set: true, size: 23, wantErr: true},
		{name: "27 rejected", set: true, size: 27, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateNodeNetmaskSize(tc.set, tc.size)
			if tc.wantErr && len(errs) != 1 {
				t.Errorf("got %v, want one error", errs)
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("got %v, want no errors", errs)
			}
		})
	}
}

// TestCreateClusterSubnetFlagSurface pins the flags create-cluster actually
// exposes, so the removal of --subnet-id and the deprecation of the alias cannot
// silently regress.
func TestCreateClusterSubnetFlagSurface(t *testing.T) {
	f := createClusterCmd.Flags()

	if f.Lookup("subnet-id") != nil {
		t.Error("--subnet-id must be gone: the CLI sends only listSubnetIds, ahead of the API dropping the deprecated subnetId field")
	}
	if f.Lookup("subnet-ids") == nil {
		t.Error("--subnet-ids must exist")
	}

	alias := f.Lookup("list-subnet-ids")
	if alias == nil {
		t.Fatal("--list-subnet-ids must remain as a deprecated alias")
	}
	if alias.Deprecated == "" {
		t.Error("--list-subnet-ids must be marked deprecated, which is what warns the caller and hides it from help")
	}
	if !alias.Hidden {
		t.Error("--list-subnet-ids must be hidden from help")
	}
}

func TestResolveSubnetIDs(t *testing.T) {
	cases := []struct {
		name          string
		subnetIDs     string
		listSubnetIDs string
		want          []string
		wantErr       string
	}{
		{
			name:      "single subnet goes through the list field",
			subnetIDs: "sub-aaa",
			want:      []string{"sub-aaa"},
		},
		{
			name:      "multiple subnets",
			subnetIDs: "sub-aaa,sub-bbb,sub-ccc",
			want:      []string{"sub-aaa", "sub-bbb", "sub-ccc"},
		},
		{
			name:      "whitespace around values is trimmed",
			subnetIDs: " sub-aaa , sub-bbb ",
			want:      []string{"sub-aaa", "sub-bbb"},
		},
		{
			name:          "deprecated alias still works alone",
			listSubnetIDs: "sub-aaa,sub-bbb",
			want:          []string{"sub-aaa", "sub-bbb"},
		},
		{
			name:          "both flags is ambiguous",
			subnetIDs:     "sub-aaa",
			listSubnetIDs: "sub-bbb",
			wantErr:       "only --subnet-ids",
		},
		{
			name:    "missing is rejected before the API call",
			wantErr: "--subnet-ids is required",
		},
		{
			name:      "commas without values is still missing",
			subnetIDs: " , ",
			wantErr:   "--subnet-ids is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveSubnetIDs(subnetFlags(tc.subnetIDs, tc.listSubnetIDs))

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil with result %v", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
