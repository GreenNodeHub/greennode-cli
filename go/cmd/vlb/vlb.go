package vlb

import (
	"github.com/greennodehub/greennode-cli/cmd/vlb/certificate"
	"github.com/greennodehub/greennode-cli/cmd/vlb/lbpackage"
	"github.com/greennodehub/greennode-cli/cmd/vlb/listener"
	"github.com/greennodehub/greennode-cli/cmd/vlb/loadbalancer"
	"github.com/greennodehub/greennode-cli/cmd/vlb/pool"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

// VlbCmd is the parent command for all VLB (load balancer) subcommands.
var VlbCmd = &cobra.Command{
	Use:   "vlb",
	Short: "VNG Load Balancer (VLB) commands",
	Long:  "Manage load balancers, listeners, pools, members and certificates.",
	// Reject unknown subcommands (nested groups don't error by default in cobra).
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	VlbCmd.AddCommand(loadbalancer.LoadBalancerCmd)
	VlbCmd.AddCommand(listener.ListenerCmd)
	VlbCmd.AddCommand(pool.PoolCmd)
	VlbCmd.AddCommand(certificate.CertificateCmd)
	VlbCmd.AddCommand(lbpackage.PackageCmd)

	// Must run after every AddCommand — it walks the assembled tree.
	registerCompletions()
	cli.RegisterService(VlbCmd)
}
