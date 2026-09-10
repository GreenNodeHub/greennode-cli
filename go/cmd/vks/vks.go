package vks

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

// VksCmd is the parent command for all VKS subcommands.
var VksCmd = &cobra.Command{
	Use:   "vks",
	Short: "GreenNode Kubernetes Service (VKS) commands",
	Long:  "Manage VKS clusters, node groups, and related resources.",
	// Reject unknown subcommands (nested groups don't error by default in cobra).
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	// Cluster commands
	VksCmd.AddCommand(listClustersCmd)
	VksCmd.AddCommand(getClusterCmd)
	VksCmd.AddCommand(createClusterCmd)
	VksCmd.AddCommand(updateClusterCmd)
	VksCmd.AddCommand(deleteClusterCmd)

	// Nodegroup commands
	VksCmd.AddCommand(listNodegroupsCmd)
	VksCmd.AddCommand(getNodegroupCmd)
	VksCmd.AddCommand(createNodegroupCmd)
	VksCmd.AddCommand(updateNodegroupCmd)
	VksCmd.AddCommand(deleteNodegroupCmd)
	VksCmd.AddCommand(updateNodegroupMetadataCmd)
	VksCmd.AddCommand(listNodesCmd)
	VksCmd.AddCommand(getNodegroupEventsCmd)
	VksCmd.AddCommand(listNodegroupImagesCmd)

	// Wait commands
	VksCmd.AddCommand(waitCmd)

	// Auto-upgrade commands
	VksCmd.AddCommand(setAutoUpgradeConfigCmd)
	VksCmd.AddCommand(deleteAutoUpgradeConfigCmd)

	// Auto-healing commands
	VksCmd.AddCommand(configAutoHealingCmd)

	// Quota commands
	VksCmd.AddCommand(getQuotaCmd)

	// Version & event commands
	VksCmd.AddCommand(listClusterVersionsCmd)
	VksCmd.AddCommand(upgradeNodegroupVersionCmd)
	VksCmd.AddCommand(getClusterEventsCmd)
	VksCmd.AddCommand(getUpgradeInsightsCmd)
	VksCmd.AddCommand(stopPOCCmd)
	VksCmd.AddCommand(registerFleetCmd)
	VksCmd.AddCommand(unregisterFleetCmd)

	// Kubeconfig commands
	VksCmd.AddCommand(generateKubeconfigCmd)
	VksCmd.AddCommand(updateKubeconfigCmd)
	VksCmd.AddCommand(acknowledgeKubeconfigWarningCmd)

	// Workspace commands
	VksCmd.AddCommand(getWorkspaceCmd)
	VksCmd.AddCommand(createWorkspaceCmd)
	VksCmd.AddCommand(resetWorkspaceServiceAccountCmd)

	cli.RegisterService(VksCmd)
	registerCompletions()
}
