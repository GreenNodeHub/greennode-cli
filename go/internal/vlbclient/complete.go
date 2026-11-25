package vlbclient

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

// listPageParams keeps completion fetches bounded — a shell completion should
// never page through thousands of resources.
var listPageParams = map[string]string{"page": "1", "size": "100"}

// buildCompleter turns a project-scoped fetch into a cobra completion function.
// Any error yields no suggestions rather than breaking the shell.
func buildCompleter(fetch func(c *client.GreennodeClient, projectID string, cmd *cobra.Command) ([]string, error)) cli.CompFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		c, cfg, err := BuildClient(cmd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		projectID, err := ProjectID(cfg)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		completions, err := fetch(c, projectID, cmd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// lbIDFromFlag reads --load-balancer-id off the command being completed; the
// nested resources cannot be listed without it.
func lbIDFromFlag(cmd *cobra.Command) string {
	id, _ := cmd.Flags().GetString("load-balancer-id")
	return id
}

// CompleteLoadBalancerIDs completes --load-balancer-id.
var CompleteLoadBalancerIDs = buildCompleter(func(c *client.GreennodeClient, projectID string, _ *cobra.Command) ([]string, error) {
	result, err := c.Get(LoadBalancers(projectID), listPageParams)
	if err != nil {
		return nil, err
	}
	return cli.ExtractCompletions(result, []string{"listData"}, "uuid", "name"), nil
})

// CompleteListenerIDs completes --listener-id, scoped to --load-balancer-id.
var CompleteListenerIDs = buildCompleter(func(c *client.GreennodeClient, projectID string, cmd *cobra.Command) ([]string, error) {
	lbID := lbIDFromFlag(cmd)
	if lbID == "" {
		return nil, nil
	}
	result, err := c.Get(Listeners(projectID, lbID), nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractCompletions(result, []string{"data"}, "uuid", "name"), nil
})

// CompletePoolIDs completes --pool-id and --default-pool-id / --redirect-pool-id,
// scoped to --load-balancer-id.
var CompletePoolIDs = buildCompleter(func(c *client.GreennodeClient, projectID string, cmd *cobra.Command) ([]string, error) {
	lbID := lbIDFromFlag(cmd)
	if lbID == "" {
		return nil, nil
	}
	result, err := c.Get(Pools(projectID, lbID), nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractCompletions(result, []string{"data"}, "uuid", "name"), nil
})

// CompletePolicyIDs completes --policy-id, scoped to --load-balancer-id and
// --listener-id.
var CompletePolicyIDs = buildCompleter(func(c *client.GreennodeClient, projectID string, cmd *cobra.Command) ([]string, error) {
	lbID := lbIDFromFlag(cmd)
	listenerID, _ := cmd.Flags().GetString("listener-id")
	if lbID == "" || listenerID == "" {
		return nil, nil
	}
	result, err := c.Get(Policies(projectID, lbID, listenerID), nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractCompletions(result, []string{"data"}, "uuid", "name"), nil
})

// CompleteCertificateIDs completes --certificate-id and the certificate flags
// on listeners.
var CompleteCertificateIDs = buildCompleter(func(c *client.GreennodeClient, projectID string, _ *cobra.Command) ([]string, error) {
	result, err := c.Get(Certificates(projectID), listPageParams)
	if err != nil {
		return nil, err
	}
	return cli.ExtractCompletions(result, []string{"listData"}, "uuid", "name"), nil
})

// CompletePackageIDs completes --package-id.
var CompletePackageIDs = buildCompleter(func(c *client.GreennodeClient, projectID string, cmd *cobra.Command) ([]string, error) {
	var params map[string]string
	if zoneID, _ := cmd.Flags().GetString("zone-id"); zoneID != "" {
		params = map[string]string{"zoneId": zoneID}
	}
	result, err := c.Get(LoadBalancers(projectID)+"/packages", params)
	if err != nil {
		return nil, err
	}
	return cli.ExtractCompletions(result, []string{"listData"}, "uuid", "name"), nil
})
