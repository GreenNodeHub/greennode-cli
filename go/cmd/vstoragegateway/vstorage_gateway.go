package vstoragegateway

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

const endpoint = "https://vmonitorapis.vngcloud.vn/vstorage-gateway"

var VStorageGatewayCmd = newVStorageGatewayCommand()

func newVStorageGatewayCommand() *cobra.Command {
	root := opengine.NewGroup("vstorage-gateway", "Inspect vStorage regions, projects, and containers")
	root.Long = `Inspect the GreenNode vMonitor-vStorage Gateway control plane.

The command group lists the documented vStorage regions, projects, and containers
available to the configured service account. All operations are read-only.`

	groups := map[string]*cobra.Command{
		"region":    opengine.NewGroup("region", "List vStorage regions"),
		"project":   opengine.NewGroup("project", "List vStorage projects"),
		"container": opengine.NewGroup("container", "List vStorage containers"),
	}
	root.AddCommand(groups["region"], groups["project"], groups["container"])
	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VStorageGatewayCmd)
}
