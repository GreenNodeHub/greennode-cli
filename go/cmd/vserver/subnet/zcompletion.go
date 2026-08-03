package subnet

import "github.com/greennodehub/greennode-cli/internal/vserverclient"

func init() {
	// --vpc-id on all subnet commands
	listCmd.RegisterFlagCompletionFunc("vpc-id", vserverclient.CompleteVPCIDs)   //nolint:errcheck
	getCmd.RegisterFlagCompletionFunc("vpc-id", vserverclient.CompleteVPCIDs)    //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("vpc-id", vserverclient.CompleteVPCIDs) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("vpc-id", vserverclient.CompleteVPCIDs) //nolint:errcheck

	// --subnet-id: resolved from --vpc-id
	getCmd.RegisterFlagCompletionFunc("subnet-id", vserverclient.CompleteSubnetIDs)    //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("subnet-id", vserverclient.CompleteSubnetIDs) //nolint:errcheck

	// create: zone
	createCmd.RegisterFlagCompletionFunc("zone-id", vserverclient.CompleteZoneIDs) //nolint:errcheck
}
