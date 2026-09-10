package vstorage

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

var VStorageCmd = newVStorageCommand()

func newVStorageCommand() *cobra.Command {
	root := opengine.NewGroup("vstorage", "Manage vStorage projects, containers, buckets, and objects")
	root.Long = `Manage GreenNode vStorage through its bearer-authenticated control plane.

HCM-3 uses the Swift-compatible container API. HAN and HCM-4 use the
Ceph/S3-compatible bucket API. Project, region, and billing commands are common
to both families. Upload and download commands generate data-plane URLs; object
bytes do not pass through this management client.`

	groups := map[string]*cobra.Command{
		"region":           opengine.NewGroup("region", "Inspect vStorage regions"),
		"project":          opengine.NewGroup("project", "Manage vStorage projects"),
		"billing":          opengine.NewGroup("billing", "Inspect vStorage quota, usage, traffic, and request statistics"),
		"container":        opengine.NewGroup("container", "Manage HCM03 Swift containers"),
		"container/object": opengine.NewGroup("object", "Manage objects and directories in an HCM03 container"),
		"bucket":           opengine.NewGroup("bucket", "Manage HAN02/HCM04 Ceph buckets"),
		"bucket/object":    opengine.NewGroup("object", "Manage objects and directories in a HAN02/HCM04 bucket"),
	}

	groups["container"].AddCommand(groups["container/object"])
	groups["bucket"].AddCommand(groups["bucket/object"])
	root.AddCommand(groups["region"], groups["project"], groups["billing"], groups["container"], groups["bucket"])

	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VStorageCmd)
}
