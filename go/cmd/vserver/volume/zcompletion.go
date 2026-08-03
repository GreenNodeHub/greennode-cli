package volume

import "github.com/greennodehub/greennode-cli/internal/vserverclient"

func init() {
	// --volume-id on commands that target an existing volume
	getCmd.RegisterFlagCompletionFunc("volume-id", vserverclient.CompleteVolumeIDs)    //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("volume-id", vserverclient.CompleteVolumeIDs) //nolint:errcheck
	resizeCmd.RegisterFlagCompletionFunc("volume-id", vserverclient.CompleteVolumeIDs) //nolint:errcheck

	// create: zone, volume type
	createCmd.RegisterFlagCompletionFunc("zone-id", vserverclient.CompleteZoneIDs)              //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("volume-type-id", vserverclient.CompleteVolumeTypeIDs) //nolint:errcheck

	// resize: volume type
	resizeCmd.RegisterFlagCompletionFunc("volume-type-id", vserverclient.CompleteVolumeTypeIDs) //nolint:errcheck
}
