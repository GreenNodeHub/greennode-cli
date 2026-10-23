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

func fetchSubnets(_ context.Context, cmd *cobra.Command) ([]string, error) {
	vpcID, _ := cmd.Flags().GetString("vpc-id")

	// Commands like create-nodegroup take --cluster-id but no --vpc-id. The
	// cluster API belongs to another service, so the lookup goes through the
	// resolver registered in the cli registry and this file keeps knowing only
	// vserver paths. Failures fall through to "no suggestions".
	if vpcID == "" {
		if clusterID, _ := cmd.Flags().GetString("cluster-id"); clusterID != "" {
			if resolved, err := cli.ResolveClusterVPC(cmd, clusterID); err == nil {
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
