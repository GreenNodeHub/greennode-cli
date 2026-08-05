package vks

import (
	"reflect"
	"testing"
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
