package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var setAutoUpgradeConfigCmd = &cobra.Command{
	Use:   "config-auto-upgrade",
	Short: "Configure auto-upgrade schedule for a cluster",
	// set-auto-upgrade-config is the former name, kept for backward compatibility.
	Aliases: []string{"set-auto-upgrade-config"},
	RunE:    runSetAutoUpgradeConfig,
}

var deleteAutoUpgradeConfigCmd = &cobra.Command{
	Use:   "delete-auto-upgrade-config",
	Short: "Delete auto-upgrade config for a cluster",
	RunE:  runDeleteAutoUpgradeConfig,
}

func init() {
	// config-auto-upgrade flags
	f := setAutoUpgradeConfigCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("weekdays", "", "Days of the week, e.g. Mon,Wed,Fri (required)")
	f.String("time", "", "Time of day in 24h format HH:mm, e.g. 03:00 (required)")
	f.Bool("dry-run", false, "Preview the auto-upgrade config without executing")
	setAutoUpgradeConfigCmd.MarkFlagRequired("cluster-id")
	setAutoUpgradeConfigCmd.MarkFlagRequired("weekdays")
	setAutoUpgradeConfigCmd.MarkFlagRequired("time")

	// delete-auto-upgrade-config flags
	g := deleteAutoUpgradeConfigCmd.Flags()
	g.String("cluster-id", "", "Cluster ID (required)")
	g.Bool("dry-run", false, "Preview what will be deleted without executing")
	g.Bool("force", false, "Skip confirmation prompt")
	deleteAutoUpgradeConfigCmd.MarkFlagRequired("cluster-id")
}

func runSetAutoUpgradeConfig(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	weekdays, _ := cmd.Flags().GetString("weekdays")
	timeVal, _ := cmd.Flags().GetString("time")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	body := map[string]interface{}{
		"weekdays": weekdays,
		"time":     timeVal,
	}

	if dryRun {
		cli.PrintDryRun("configure", fmt.Sprintf("auto-upgrade for cluster %s", clusterID), body)
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(
		fmt.Sprintf("/v1/clusters/%s/auto-upgrade-config", clusterID), body,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}

func runDeleteAutoUpgradeConfig(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	fmt.Println("The following will be deleted:")
	fmt.Printf("  Auto-upgrade config for cluster: %s\n", clusterID)
	fmt.Println("\nThis action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}

	if !cli.Confirm(force, "Are you sure you want to delete the auto-upgrade config?") {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Delete(
		fmt.Sprintf("/v1/clusters/%s/auto-upgrade-config", clusterID), nil,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
