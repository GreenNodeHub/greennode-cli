package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateSecgroupCmd = &cobra.Command{
	Use:   "update-secgroup",
	Short: "Update the security groups attached to a vServer instance",
	Long: `Replace the set of security groups attached to a vServer instance.

The provided security group IDs become the complete list for the server —
any security group not included is detached. Run 'vserver secgroup list' to
see available security groups.`,
	RunE: runUpdateSecgroup,
}

func init() {
	f := updateSecgroupCmd.Flags()
	f.String("server-id", "", "Server ID (required)")
	f.String("security-group", "", "Security group IDs to attach (comma-separated, required)")

	for _, name := range []string{"server-id", "security-group"} {
		if err := updateSecgroupCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runUpdateSecgroup(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	securityGroup, _ := cmd.Flags().GetString("security-group")

	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}

	secgroupIDs := parseCommaSeparated(securityGroup)
	if len(secgroupIDs) == 0 {
		return fmt.Errorf("at least one security group ID is required (--security-group)")
	}
	for _, id := range secgroupIDs {
		if err := validator.ValidateID(id, "security-group"); err != nil {
			return err
		}
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(
		fmt.Sprintf("/v2/%s/servers/%s/update-sec-group", projectID, serverID),
		map[string]interface{}{"securityGroup": secgroupIDs},
	)
	if err != nil {
		return fmt.Errorf("failed to update security groups for server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, transformServerResult(result))
}
