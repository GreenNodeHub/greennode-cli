package networkinterface

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new elastic network interface",
	Long: `Create a new elastic network interface in a zone.

Tags are optional key/value pairs. Pass --tag once per pair, in key=value form,
for example: --tag env=prod --tag vks-cluster-ids=k8s-...`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "Name of the network interface (required)")
	f.String("zone-id", "", "Availability zone ID (required)")
	f.StringArray("tag", nil, "Tag to attach, in key=value form (repeatable)")

	for _, name := range []string{"name", "zone-id"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	if strings.TrimSpace(zoneID) == "" {
		return fmt.Errorf("--zone-id is required")
	}

	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"name":   name,
		"tags":   tags,
		"zoneId": zoneID,
	}

	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/network-interfaces-elastic", projectID), body)
	if err != nil {
		return fmt.Errorf("failed to create network interface: %w", err)
	}

	return outputResult(cmd, cfg, result)
}
