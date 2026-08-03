package secgroup

import "github.com/greennodehub/greennode-cli/internal/vserverclient"

func init() {
	getCmd.RegisterFlagCompletionFunc("secgroup-id", vserverclient.CompleteSecgroupIDs)    //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("secgroup-id", vserverclient.CompleteSecgroupIDs) //nolint:errcheck
}
