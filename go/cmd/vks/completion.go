package vks

import (
	"context"
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

func fetchClusterIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	c, err := createClient(cmd)
	if err != nil {
		return nil, err
	}
	res, err := c.Get("/v1/clusters", map[string]string{"page": "0", "pageSize": "100"})
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(res, "id"), nil
}

func fetchNodegroupIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if clusterID == "" {
		return nil, nil
	}
	c, err := createClient(cmd)
	if err != nil {
		return nil, err
	}
	res, err := c.Get(fmt.Sprintf("/v1/clusters/%s/node-groups", clusterID), map[string]string{"page": "0", "pageSize": "100"})
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(res, "id"), nil
}

func fetchK8sVersions(_ context.Context, cmd *cobra.Command) ([]string, error) {
	c, err := createClient(cmd)
	if err != nil {
		return nil, err
	}
	res, err := c.Get("/v1/cluster-versions", nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(res, "version", "name", "id"), nil
}

func flagCompleters() map[string]cli.CompFunc {
	return map[string]cli.CompFunc{
		"cluster-id":             cli.FlagFromAPI(fetchClusterIDs),
		"nodegroup-id":           cli.FlagFromAPI(fetchNodegroupIDs),
		"k8s-version":            cli.FlagFromAPI(fetchK8sVersions),
		"os":                     cli.FlagValues("ubuntu", "linux", "rocky"),
		"network-type":           cli.FlagValues("TIGERA", "CILIUM_OVERLAY", "CILIUM_NATIVE_ROUTING"),
		"release-channel":        cli.FlagValues("RAPID", "STABLE"),
		"private-cluster":        cli.FlagValues("enabled", "disabled"),
		"private-nodes":          cli.FlagValues("enabled", "disabled"),
		"load-balancer-plugin":   cli.FlagValues("enabled", "disabled"),
		"block-store-csi-plugin": cli.FlagValues("enabled", "disabled"),
		"vpc-id":                 cli.ResourceCompletion("vserver:network"),
		"subnet-id":              cli.ResourceCompletion("vserver:subnet"),
		"subnet-ids":             cli.ResourceCompletion("vserver:subnet"),
		"ssh-key-id":             cli.ResourceCompletion("vserver:sshkey"),
		"security-groups":        cli.ResourceCompletion("vserver:secgroup"),
		"disk-type":              cli.ResourceCompletion("vserver:volumetype"),
	}
}

// resolveClusterVPC returns the ID of the VPC a cluster lives in. Registered into
// the cli registry so vserver-owned subnet completion works on VKS commands that
// take --cluster-id but no --vpc-id (create-nodegroup), without that package
// needing to know VKS API paths.
func resolveClusterVPC(cmd *cobra.Command, clusterID string) (string, error) {
	c, err := createClient(cmd)
	if err != nil {
		return "", err
	}
	resp, err := c.Get(fmt.Sprintf("/v1/clusters/%s", clusterID), nil)
	if err != nil {
		return "", err
	}
	m, ok := resp.(map[string]interface{})
	if !ok {
		return "", nil
	}
	vpcID, _ := m["vpcId"].(string)
	return vpcID, nil
}

// registerCompletions attaches value completers to every VKS subcommand that
// declares a matching flag. Called from vks.go init() after AddCommand.
func registerCompletions() {
	cli.RegisterClusterVPCResolver(resolveClusterVPC)

	completers := flagCompleters()
	for _, c := range VksCmd.Commands() {
		for name, fn := range completers {
			if c.Flags().Lookup(name) != nil {
				_ = c.RegisterFlagCompletionFunc(name, fn)
			}
		}
	}
}
