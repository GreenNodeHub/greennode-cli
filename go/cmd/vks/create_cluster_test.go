package vks

import (
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

// TestCreateClusterSubnetFlagSurface pins the flags create-cluster actually
// exposes, so the removal of --subnet-id and the deprecation of the alias cannot
// silently regress.
func TestCreateClusterSubnetFlagSurface(t *testing.T) {
	f := createClusterCmd.Flags()

	if f.Lookup("subnet-id") != nil {
		t.Error("--subnet-id must be gone: the API has no single-subnet subnetId field")
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
