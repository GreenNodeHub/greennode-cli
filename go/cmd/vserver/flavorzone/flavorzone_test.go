package flavorzone

import "testing"

func TestFlavorZoneCommandUsesRunE(t *testing.T) {
	if FlavorZoneCmd.Run != nil || FlavorZoneCmd.RunE == nil {
		t.Fatal("flavor-zone must use RunE")
	}
}
