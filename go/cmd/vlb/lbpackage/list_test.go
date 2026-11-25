package lbpackage

import "testing"

func TestCheckZoneWithAll(t *testing.T) {
	for _, tc := range []struct {
		all, zoneIDSet, wantErr bool
	}{
		{all: true, zoneIDSet: true, wantErr: true},
		// `--all=false --zone-id Z` (e.g. --all=$ALL in a script) is a plain zone list.
		{all: false, zoneIDSet: true},
		{all: true, zoneIDSet: false},
		{all: false, zoneIDSet: false},
	} {
		if err := checkZoneWithAll(tc.all, tc.zoneIDSet); (err != nil) != tc.wantErr {
			t.Errorf("checkZoneWithAll(all=%v, zoneIDSet=%v) error = %v, wantErr %v",
				tc.all, tc.zoneIDSet, err, tc.wantErr)
		}
	}
}
