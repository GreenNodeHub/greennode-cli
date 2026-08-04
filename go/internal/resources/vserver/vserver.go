// Package vserver registers shell-completion providers for vserver-owned
// resources (VPC, subnet, SSH key, security group, volume type) into the shared
// cli registry. Platform-owned for now; a future vserver CLI can re-register
// these keys without changing consumers (e.g. vks).
//
// Coupling note: this file knows vserver API *paths/response shapes* (not the
// vserver CLI). It is the single isolated place that does.
package vserver

import (
	"context"
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	for key, fn := range registeredKeys() {
		cli.RegisterResourceCompleter(key, fn)
	}
}

func listPath(projectID, tmpl string) string {
	return fmt.Sprintf(tmpl, projectID)
}

func subnetPath(projectID, vpcID string) (string, bool) {
	if projectID == "" || vpcID == "" {
		return "", false
	}
	return fmt.Sprintf("/v2/%s/networks/%s/subnets", projectID, vpcID), true
}

func projectID(cmd *cobra.Command) string {
	profile, _ := cmd.Flags().GetString("profile")
	cfg, err := config.LoadConfig(profile)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.ProjectID
}

func simpleList(tmpl string, idFields ...string) func(context.Context, *cobra.Command) ([]string, error) {
	return func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		proj := projectID(cmd)
		if proj == "" {
			return nil, nil
		}
		c, err := cli.NewClient(cmd, "vserver")
		if err != nil {
			return nil, err
		}
		res, err := c.Get(listPath(proj, tmpl), nil)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(res, idFields...), nil
	}
}

// resolveVPCFromCluster fetches a cluster by ID and returns its VPC ID.
func resolveVPCFromCluster(cmd *cobra.Command, clusterID string) (string, error) {
	c, err := cli.NewClient(cmd, "vks")
	if err != nil {
		return "", err
	}
	resp, err := c.Get(fmt.Sprintf("/v1/clusters/%s", clusterID), nil)
	if err != nil {
		return "", err
	}
	// Response is map[string]interface{}, extract vpcId.
	m, ok := resp.(map[string]interface{})
	if !ok {
		return "", nil
	}
	vpcID, ok := m["vpcId"].(string)
	if !ok {
		return "", nil
	}
	return vpcID, nil
}

func fetchSubnets(_ context.Context, cmd *cobra.Command) ([]string, error) {
	vpcID, _ := cmd.Flags().GetString("vpc-id")

	// If --vpc-id is missing but --cluster-id is present, resolve the VPC from the cluster.
	// This makes completion work for commands like create-nodegroup that don't take --vpc-id.
	if vpcID == "" {
		if clusterID, _ := cmd.Flags().GetString("cluster-id"); clusterID != "" {
			if resolved, err := resolveVPCFromCluster(cmd, clusterID); err == nil && resolved != "" {
				vpcID = resolved
			}
		}
	}

	path, ok := subnetPath(projectID(cmd), vpcID)
	if !ok {
		return nil, nil
	}
	c, err := cli.NewClient(cmd, "vserver")
	if err != nil {
		return nil, err
	}
	res, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(res, "uuid", "id"), nil
}

func registeredKeys() map[string]cli.CompFunc {
	return map[string]cli.CompFunc{
		"vserver:network":    cli.FlagFromAPI(simpleList("/v2/%s/networks", "id")),
		"vserver:subnet":     cli.FlagFromAPI(fetchSubnets),
		"vserver:sshkey":     cli.FlagFromAPI(simpleList("/v2/%s/sshKeys", "id", "name")),
		"vserver:secgroup":   cli.FlagFromAPI(simpleList("/v2/%s/secgroups", "id")),
		"vserver:volumetype": cli.FlagFromAPI(simpleList("/v1/%s/volume_types", "id")),
	}
}
