package vpc

import "github.com/greennodehub/greennode-cli/internal/vserverclient"

func init() {
	getCmd.RegisterFlagCompletionFunc("vpc-id", vserverclient.CompleteVPCIDs)    //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("vpc-id", vserverclient.CompleteVPCIDs) //nolint:errcheck
}
