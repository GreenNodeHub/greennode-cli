// Package vbackup implements GreenNode vBackup Gateway management commands.
package vbackup

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

// VBackupCmd is the parent command for vBackup Gateway operations.
var VBackupCmd = newVBackupCommand()

func newVBackupCommand() *cobra.Command {
	root := opengine.NewGroup("vbackup", "Manage HCM-3 vServer backup policies, servers, and restore history")
	root.Long = `Manage GreenNode vBackup Gateway resources in HCM-3.

The command group covers the published backup gateway control plane for vServer
backup policies, protected servers, restore points, destinations, and history.
Use --dry-run to preview changes without accessing credentials or the API.`

	groups := map[string]*cobra.Command{
		"backend":       opengine.NewGroup("backend", "Inspect vBackup backends"),
		"vserver":       opengine.NewGroup("vserver", "Inspect or create vServer backup resources"),
		"destination":   opengine.NewGroup("destination", "List backup destinations"),
		"policy":        opengine.NewGroup("policy", "Manage backup policies"),
		"server":        opengine.NewGroup("server", "Manage backup servers"),
		"configuration": opengine.NewGroup("configuration", "Inspect vBackup configuration"),
		"history":       opengine.NewGroup("history", "Inspect backup and restoration history"),
		"volume":        opengine.NewGroup("volume", "Inspect volume backup usage"),
	}
	root.AddCommand(groups["backend"], groups["vserver"], groups["destination"], groups["policy"], groups["server"], groups["configuration"], groups["history"], groups["volume"])
	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VBackupCmd)
}
