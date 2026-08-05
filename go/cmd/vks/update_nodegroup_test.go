package vks

import (
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
			name: "maxSurge only fills maxUnavailable",
			in:   map[string]interface{}{"maxSurge": 2},
			want: map[string]interface{}{"maxSurge": 2, "maxUnavailable": 0},
		},
		{
			name: "both missing fill both",
			in:   map[string]interface{}{},
			want: map[string]interface{}{"maxSurge": 1, "maxUnavailable": 0},
		},
		{
			name: "both null fill both",
			in:   map[string]interface{}{"maxSurge": nil, "maxUnavailable": nil},
			want: map[string]interface{}{"maxSurge": 1, "maxUnavailable": 0},
		},
		{
			name: "both set unchanged",
			in:   map[string]interface{}{"maxSurge": 3, "maxUnavailable": 1},
			want: map[string]interface{}{"maxSurge": 3, "maxUnavailable": 1},
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
