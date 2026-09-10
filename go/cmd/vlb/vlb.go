package vlb

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

var VLBCmd = newVLBCommand()

func newVLBCommand() *cobra.Command {
	root := opengine.NewGroup("vlb", "Manage vLB load balancers and traffic resources")
	root.Long = `Manage GreenNode vLB resources through the project-scoped vLB API.

The vLB API is available in HCM-3 and HAN. Every operation requires an
explicit project ID. Mutations accept exact official-schema JSON through
--body and support an offline --dry-run preview.`

	groups := map[string]*cobra.Command{
		"certificate":   opengine.NewGroup("certificate", "Manage TLS certificates"),
		"load-balancer": opengine.NewGroup("load-balancer", "Manage load balancers and inspect service metadata"),
		"listener":      opengine.NewGroup("listener", "Manage load-balancer listeners"),
		"l7-policy":     opengine.NewGroup("l7-policy", "Manage listener L7 policies"),
		"pool":          opengine.NewGroup("pool", "Manage load-balancer pools and members"),
	}
	root.AddCommand(groups["certificate"], groups["load-balancer"], groups["listener"], groups["l7-policy"], groups["pool"])

	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VLBCmd)
}
