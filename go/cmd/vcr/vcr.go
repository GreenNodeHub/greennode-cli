package vcr

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

const endpoint = "https://vcr.api.vngcloud.vn"

var VCRCmd = newVCRCommand()

func newVCRCommand() *cobra.Command {
	root := opengine.NewGroup("vcr", "Manage vCR repositories, images, artifacts, and repository users")
	root.Long = `Manage GreenNode vCR through its global registry management API.

The vCR client uses configured IAM credentials and a
single global endpoint. Repository, image, artifact, and repository-user
commands preserve the API's documented page and size parameters without
inventing cross-page aggregation.`

	groups := map[string]*cobra.Command{
		"repository": opengine.NewGroup("repository", "Manage container registry repositories"),
		"image":      opengine.NewGroup("image", "Inspect and delete repository images"),
		"artifact":   opengine.NewGroup("artifact", "Inspect and delete image artifacts"),
		"user":       opengine.NewGroup("user", "Manage repository users and permissions"),
	}
	root.AddCommand(groups["repository"], groups["image"], groups["artifact"], groups["user"])

	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VCRCmd)
}
