package server

import "github.com/greennodehub/greennode-cli/internal/vserverclient"

func init() {
	// --server-id on all commands that target an existing server
	getCmd.RegisterFlagCompletionFunc("server-id", vserverclient.CompleteServerIDs)    //nolint:errcheck
	startCmd.RegisterFlagCompletionFunc("server-id", vserverclient.CompleteServerIDs)  //nolint:errcheck
	stopCmd.RegisterFlagCompletionFunc("server-id", vserverclient.CompleteServerIDs)   //nolint:errcheck
	rebootCmd.RegisterFlagCompletionFunc("server-id", vserverclient.CompleteServerIDs) //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("server-id", vserverclient.CompleteServerIDs) //nolint:errcheck
	resizeCmd.RegisterFlagCompletionFunc("server-id", vserverclient.CompleteServerIDs) //nolint:errcheck

	// create: zone, network, subnet, image, volume types, security group
	createCmd.RegisterFlagCompletionFunc("zone-id", vserverclient.CompleteZoneIDs)                 //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("network-id", vserverclient.CompleteVPCIDs)               //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("subnet-id", vserverclient.CompleteSubnetIDs)             //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("image-id", vserverclient.CompleteImageIDs)               //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("root-disk-type-id", vserverclient.CompleteVolumeTypeIDs) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("data-disk-type-id", vserverclient.CompleteVolumeTypeIDs) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("security-group", vserverclient.CompleteSecgroupIDs)      //nolint:errcheck
}
