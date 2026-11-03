package backupstorage

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the backup storage you own",
	Long: "List your backup storage: its quota, how much of it your backups occupy, and " +
		"which package it came from.\n\n" +
		"This endpoint takes no pagination or filters.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var listPackagesCmd = &cobra.Command{
	Use:   "list-packages",
	Short: "List the backup storage packages available to buy",
	Long: "List the purchasable backup storage packages, with their quota and price.\n\n" +
		"The packageId is what --package-id expects on 'backup-storage create' and " +
		"'backup-storage resize'.",
	Args: cobra.NoArgs,
	RunE: runListPackages,
}

var packageColumns = []string{
	"engineGroup", "packageId", "packageName", "packageQuota", "price", "sku", "description",
}

func runList(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// Note the path: the listing of what you OWN is /backup-storages/information,
	// while /backup-storages (see list-packages) is the catalogue of what you can buy.
	result, err := apiClient.Get(basePath+"/information", nil)
	if err != nil {
		return fmt.Errorf("failed to list backup storage: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, storageColumns)
}

func runListPackages(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(basePath, nil)
	if err != nil {
		return fmt.Errorf("failed to list backup storage packages: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, flattenPackages(vdbclient.Unwrap(result)), packageColumns)
}

// flattenPackages turns the response's two levels into one row per package.
//
// The payload is an array of {engineGroup, packages[]}, so a table would otherwise
// show one row per engine group with the packages collapsed into a Go-printed slice —
// and OutputWithColumns picks the first array it finds, which is the outer one.
// Flattening keeps engineGroup on each row so the grouping is not lost.
func flattenPackages(payload interface{}) interface{} {
	groups, ok := payload.([]interface{})
	if !ok {
		return payload
	}

	var rows []interface{}
	for _, item := range groups {
		group, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		packages, _ := group["packages"].([]interface{})
		for _, entry := range packages {
			pkg, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			row := make(map[string]interface{}, len(pkg)+1)
			for k, v := range pkg {
				row[k] = v
			}
			row["engineGroup"] = group["engineGroup"]
			rows = append(rows, row)
		}
	}
	if rows == nil {
		// Nothing recognisable: hand back the original so JSON output is unchanged and
		// the user can see what the API actually said.
		return payload
	}
	return rows
}
