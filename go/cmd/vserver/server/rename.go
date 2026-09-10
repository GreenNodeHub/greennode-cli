package server

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{Use: "rename", Short: "Rename a server", RunE: runRename}

func init() {
	f := renameCmd.Flags()
	f.String("server-id", "", "Server ID (required)")
	f.String("name", "", "New server name (required)")
	f.String("zone-id", "", "Availability zone ID")
	f.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	f.Bool("dry-run", false, "Validate the rename without executing")
	for _, name := range []string{"server-id", "name"} {
		if err := renameCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runRename(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	name, _ := cmd.Flags().GetString("name")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice("rename")
		return nil
	}
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	body := map[string]any{"newName": name, "tags": tags, "zoneId": nilIfEmpty(zoneID)}
	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/servers/%s/rename", projectID, serverID), body)
	if err != nil {
		return fmt.Errorf("failed to rename server %s: %w", serverID, err)
	}
	return outputServerDetail(cmd, cfg, result)
}
