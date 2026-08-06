package vks

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestUpgradeConfigWithDefaults(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]interface{}
		want map[string]interface{}
	}{
		{
			// strategy is required by NodeGroupUpgradeConfigDto: omitting it is a
			// 400, so a partial config must gain it.
			name: "maxSurge only fills maxUnavailable and strategy",
			in:   map[string]interface{}{"maxSurge": 2},
			want: map[string]interface{}{"maxSurge": 2, "maxUnavailable": 0, "strategy": "SURGE"},
		},
		{
			name: "all missing fill all",
			in:   map[string]interface{}{},
			want: map[string]interface{}{"maxSurge": 1, "maxUnavailable": 0, "strategy": "SURGE"},
		},
		{
			name: "all null fill all",
			in:   map[string]interface{}{"maxSurge": nil, "maxUnavailable": nil, "strategy": nil},
			want: map[string]interface{}{"maxSurge": 1, "maxUnavailable": 0, "strategy": "SURGE"},
		},
		{
			name: "empty strategy filled",
			in:   map[string]interface{}{"strategy": ""},
			want: map[string]interface{}{"maxSurge": 1, "maxUnavailable": 0, "strategy": "SURGE"},
		},
		{
			name: "all set unchanged",
			in:   map[string]interface{}{"maxSurge": 3, "maxUnavailable": 1, "strategy": "SURGE"},
			want: map[string]interface{}{"maxSurge": 3, "maxUnavailable": 1, "strategy": "SURGE"},
		},
		{
			name: "strategy preserved, nulls filled",
			in:   map[string]interface{}{"strategy": "SURGE", "maxSurge": nil},
			want: map[string]interface{}{"strategy": "SURGE", "maxSurge": 1, "maxUnavailable": 0},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := upgradeConfigWithDefaults(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func autoScaleFlags(autoScale string, disable bool) *pflag.FlagSet {
	f := pflag.NewFlagSet("update-nodegroup", pflag.ContinueOnError)
	f.String("auto-scale", autoScale, "")
	f.Bool("disable-auto-scale", disable, "")
	return f
}

func TestResolveAutoScaleConfig(t *testing.T) {
	cases := []struct {
		name      string
		autoScale string
		disable   bool
		wantValue interface{}
		wantSet   bool
		wantErr   string
	}{
		{
			name:    "disabled sends nil",
			disable: true,
			wantSet: true,
		},
		{
			name:    "omitted returns unset",
			wantSet: false,
		},
		{
			name:      "both flags is error",
			autoScale: "minSize=2,maxSize=10",
			disable:   true,
			wantErr:   "cannot use --auto-scale and --disable-auto-scale together",
		},
		{
			name:      "shorthand both fields",
			autoScale: "minSize=2,maxSize=10",
			wantValue: map[string]interface{}{"minSize": 2, "maxSize": 10},
			wantSet:   true,
		},
		{
			name:      "json both fields",
			autoScale: `{"minSize":2,"maxSize":10}`,
			wantValue: map[string]interface{}{"minSize": float64(2), "maxSize": float64(10)},
			wantSet:   true,
		},
		{
			name:      "shorthand missing maxSize",
			autoScale: "minSize=2",
			wantErr:   "both minSize and maxSize are required",
		},
		{
			name:      "empty object missing both",
			autoScale: "{}",
			wantErr:   "both minSize and maxSize are required",
		},
		{
			name:      "json minSize null",
			autoScale: `{"minSize":null,"maxSize":5}`,
			wantErr:   "minSize must be an integer",
		},
		{
			name:      "json minSize non-integral",
			autoScale: `{"minSize":2.5,"maxSize":5}`,
			wantErr:   "minSize must be an integer",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, set, err := resolveAutoScaleConfig(autoScaleFlags(tc.autoScale, tc.disable))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil value=%v set=%v", tc.wantErr, got, set)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if set != tc.wantSet {
				t.Fatalf("set = %v, want %v", set, tc.wantSet)
			}
			if tc.wantSet && !reflect.DeepEqual(got, tc.wantValue) {
				t.Errorf("value = %v, want %v", got, tc.wantValue)
			}
		})
	}
}

func updateFlags(numNodes, securityGroups, autoScale, upgradeConfig string, disable bool) *pflag.FlagSet {
	f := pflag.NewFlagSet("update-nodegroup", pflag.ContinueOnError)
	f.String("num-nodes", numNodes, "")
	f.String("security-groups", securityGroups, "")
	f.String("auto-scale", autoScale, "")
	f.String("upgrade-config", upgradeConfig, "")
	f.Bool("disable-auto-scale", disable, "")
	return f
}

func TestBuildUpdateNodegroupBody(t *testing.T) {
	t.Run("disable only sends null", func(t *testing.T) {
		body, err := buildUpdateNodegroupBody(updateFlags("", "", "", "", true))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]interface{}{"autoScaleConfig": nil}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("got %v, want %v", body, want)
		}
		js, _ := json.Marshal(body)
		if !strings.Contains(string(js), `"autoScaleConfig":null`) {
			t.Errorf("json = %s, want \"autoScaleConfig\":null present", js)
		}
	})

	t.Run("auto-scale valid object sets autoScaleConfig", func(t *testing.T) {
		body, err := buildUpdateNodegroupBody(updateFlags("", "", "minSize=2,maxSize=10", "", false))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		asc, ok := body["autoScaleConfig"].(map[string]interface{})
		if !ok {
			t.Fatalf("autoScaleConfig missing or wrong type: %v", body["autoScaleConfig"])
		}
		if asc["minSize"] != 2 || asc["maxSize"] != 10 {
			t.Errorf("autoScaleConfig = %v, want minSize=2 maxSize=10", asc)
		}
	})

	t.Run("upgrade fills defaults including required strategy", func(t *testing.T) {
		body, err := buildUpdateNodegroupBody(updateFlags("", "", "", "maxSurge=2", false))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		uc, ok := body["upgradeConfig"].(map[string]interface{})
		if !ok {
			t.Fatalf("upgradeConfig missing or wrong type: %v", body["upgradeConfig"])
		}
		if uc["maxSurge"] != 2 || uc["maxUnavailable"] != 0 {
			t.Errorf("got %v, want maxSurge=2 maxUnavailable=0", uc)
		}
		// NodeGroupUpgradeConfigDto marks strategy required; sending
		// {"maxSurge":2,"maxUnavailable":0} alone is rejected by the API.
		if uc["strategy"] != "SURGE" {
			t.Errorf("strategy = %v, want SURGE", uc["strategy"])
		}
	})

	t.Run("upgrade bounds surface as error", func(t *testing.T) {
		_, err := buildUpdateNodegroupBody(updateFlags("", "", "", "maxSurge=0", false))
		if err == nil || !strings.Contains(err.Error(), "maxSurge must be between 1 and 100") {
			t.Errorf("err = %v, want maxSurge bound error", err)
		}
	})

	t.Run("num-nodes must be an integer", func(t *testing.T) {
		_, err := buildUpdateNodegroupBody(updateFlags("abc", "", "", "", false))
		if err == nil || !strings.Contains(err.Error(), "--num-nodes must be an integer") {
			t.Errorf("err = %v, want integer parse error (a swallowed error would scale to 0 nodes)", err)
		}
	})

	t.Run("num-nodes only", func(t *testing.T) {
		body, err := buildUpdateNodegroupBody(updateFlags("3", "", "", "", false))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]interface{}{"numNodes": 3}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("got %v, want %v", body, want)
		}
	})

	t.Run("auto-scale validation surfaces as error", func(t *testing.T) {
		_, err := buildUpdateNodegroupBody(updateFlags("", "", "minSize=2", "", false))
		if err == nil || !strings.Contains(err.Error(), "both minSize and maxSize are required") {
			t.Errorf("err = %v, want to contain validation message", err)
		}
	})

	t.Run("nothing to update", func(t *testing.T) {
		_, err := buildUpdateNodegroupBody(updateFlags("", "", "", "", false))
		if err == nil || !strings.Contains(err.Error(), "nothing to update") {
			t.Errorf("err = %v, want nothing-to-update error", err)
		}
	})
}

// TestUpdateNodegroupFlagSurface pins the flags update-nodegroup exposes. The
// body composer reads flags by name and ignores lookup errors, so without this a
// renamed flag would silently degrade to a zero value with every test still green.
func TestUpdateNodegroupFlagSurface(t *testing.T) {
	f := updateNodegroupCmd.Flags()
	for _, name := range []string{"num-nodes", "security-groups", "auto-scale", "disable-auto-scale", "upgrade-config"} {
		if f.Lookup(name) == nil {
			t.Errorf("--%s must exist on update-nodegroup", name)
		}
	}
}

func TestUpgradeConfigWithDefaultsDoesNotMutateInput(t *testing.T) {
	in := map[string]interface{}{"maxSurge": 2}
	upgradeConfigWithDefaults(in)
	if !reflect.DeepEqual(in, map[string]interface{}{"maxSurge": 2}) {
		t.Errorf("input was mutated: %v", in)
	}
}

func TestValidateUpgradeConfigObject(t *testing.T) {
	cases := []struct {
		name    string
		in      map[string]interface{}
		wantErr string
	}{
		{
			name: "defaults are valid",
			in:   map[string]interface{}{"strategy": "SURGE", "maxSurge": 1, "maxUnavailable": 0},
		},
		{
			name: "json floats accepted",
			in:   map[string]interface{}{"strategy": "SURGE", "maxSurge": float64(5), "maxUnavailable": float64(10)},
		},
		{
			name:    "maxSurge below minimum",
			in:      map[string]interface{}{"strategy": "SURGE", "maxSurge": 0},
			wantErr: "maxSurge must be between 1 and 100",
		},
		{
			name:    "maxSurge above maximum",
			in:      map[string]interface{}{"strategy": "SURGE", "maxSurge": 101},
			wantErr: "maxSurge must be between 1 and 100",
		},
		{
			name:    "maxUnavailable above maximum",
			in:      map[string]interface{}{"strategy": "SURGE", "maxUnavailable": 101},
			wantErr: "maxUnavailable must be between 0 and 100",
		},
		{
			name:    "non-integral maxSurge",
			in:      map[string]interface{}{"strategy": "SURGE", "maxSurge": 2.5},
			wantErr: "maxSurge must be an integer",
		},
		{
			name:    "strategy wrong type",
			in:      map[string]interface{}{"strategy": 5},
			wantErr: "strategy must be a non-empty string",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUpgradeConfigObject(tc.in)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateAutoScaleObjectBounds(t *testing.T) {
	cases := []struct {
		name    string
		in      map[string]interface{}
		wantErr string
	}{
		{
			name: "zero to one is valid",
			in:   map[string]interface{}{"minSize": 0, "maxSize": 1},
		},
		{
			name:    "negative minSize",
			in:      map[string]interface{}{"minSize": -1, "maxSize": 5},
			wantErr: "minSize must be 0 or greater",
		},
		{
			name:    "maxSize below minimum",
			in:      map[string]interface{}{"minSize": 0, "maxSize": 0},
			wantErr: "maxSize must be 1 or greater",
		},
		{
			name:    "inverted range",
			in:      map[string]interface{}{"minSize": 5, "maxSize": 2},
			wantErr: "minSize (5) must not exceed maxSize (2)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAutoScaleObject(tc.in)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseNumNodes(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr string
	}{
		{in: "3", want: 3},
		{in: "0", want: 0},
		{in: " 7 ", want: 7},
		{in: "abc", wantErr: "must be an integer"},
		{in: "3.5", wantErr: "must be an integer"},
		{in: "3abc", wantErr: "must be an integer"},
		{in: "-1", wantErr: "must be 0 or greater"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseNumNodes(tc.in)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}
