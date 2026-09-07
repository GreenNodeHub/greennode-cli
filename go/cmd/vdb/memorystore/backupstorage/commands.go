package backupstorage

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the backup storage you own",
	Long: "List your MemoryStore backup storage: its quota, how much of it your backups " +
		"occupy, and which package it came from.\n\n" +
		"This reads GET /backup-storages — the same path that, in the Relational Database " +
		"API, lists the packages available to BUY. Here that catalogue is " +
		"'list-packages'.",
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

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Buy backup storage",
	Long: "Buy a backup storage package.\n\n" +
		"THIS COSTS MONEY: order/payment flow, asynchronous. Run 'backup-storage " +
		"list-packages' for the package IDs, and 'backup-storage list' to see whether you " +
		"already own storage — growing what you have is 'backup-storage resize'.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Change the package of existing backup storage",
	Long: "Move existing backup storage to a different package, usually a larger one.\n\n" +
		"THIS COSTS MONEY: order/payment flow. Shrinking below what your backups already " +
		"occupy is rejected by the API — check the usage column in 'backup-storage list'.",
	Args: cobra.NoArgs,
	RunE: runResize,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Release backup storage",
	Long: "Give up a backup storage quota.\n\n" +
		"Any backup that relies on this storage is at risk: release it only when your " +
		"backups fit in the free allowance, or after deleting them.\n\n" +
		"Note the endpoint is /actions/delete, singular — the relational one is " +
		"/actions/deletions.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	c := createCmd.Flags()
	c.String("package-id", "", "Backup storage package to buy (required; see 'backup-storage list-packages')")
	c.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	c.Bool("force", false, "Skip the confirmation prompt")
	createCmd.MarkFlagRequired("package-id") //nolint:errcheck

	r := resizeCmd.Flags()
	r.String("storage-id", "", "Backup storage to change (required; see 'backup-storage list')")
	r.String("package-id", "", "New package (required)")
	r.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	r.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	r.Bool("force", false, "Skip the confirmation prompt")
	resizeCmd.MarkFlagRequired("storage-id") //nolint:errcheck
	resizeCmd.MarkFlagRequired("package-id") //nolint:errcheck

	d := deleteCmd.Flags()
	d.String("storage-id", "", "Backup storage to release (required)")
	d.Bool("dry-run", false, "Print the request that would be sent without releasing anything")
	d.Bool("force", false, "Skip the confirmation prompt")
	deleteCmd.MarkFlagRequired("storage-id") //nolint:errcheck

	// Bound here, next to the flags: see the init-order note in completion.go.
	createCmd.RegisterFlagCompletionFunc("package-id", packageIDCompletion()) //nolint:errcheck
	resizeCmd.RegisterFlagCompletionFunc("storage-id", storageIDCompletion()) //nolint:errcheck
	resizeCmd.RegisterFlagCompletionFunc("package-id", packageIDCompletion()) //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("storage-id", storageIDCompletion()) //nolint:errcheck
}

func runList(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(basePath, nil)
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

	result, err := apiClient.Get(basePath+"/packages", nil)
	if err != nil {
		return fmt.Errorf("failed to list backup storage packages: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, flattenPackages(vdbclient.Unwrap(result)), packageColumns)
}

// flattenPackages turns the response's two levels into one row per package. The
// payload is an array of {engineGroup, packages[]}, and table output picks the first
// array it finds — the outer one — so each package becomes its own row, keeping its
// engine group.
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
		return payload
	}
	return rows
}

func runCreate(cmd *cobra.Command, args []string) error {
	packageID, _ := cmd.Flags().GetString("package-id")

	body := map[string]interface{}{"backupPackageId": packageID}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("buy", fmt.Sprintf("backup storage package %s", packageID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Buy backup storage package %s? This places a paid order.", packageID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(paymentPath, body)
	if err != nil {
		return fmt.Errorf("failed to buy backup storage package %s: %w", packageID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runResize(cmd *cobra.Command, args []string) error {
	storageID, _ := cmd.Flags().GetString("storage-id")
	if err := requireStorageID(storageID); err != nil {
		return err
	}

	packageID, _ := cmd.Flags().GetString("package-id")
	poc, _ := cmd.Flags().GetBool("poc")

	body := vdbclient.ResizeBody(vdbclient.ResourceTypeBackupStorage, storageID, "resize",
		map[string]interface{}{
			"backupPackageId": packageID,
			"poc":             poc,
		})

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("resize", fmt.Sprintf("backup storage %s", storageID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Move backup storage %s to package %s? This places a paid order.", storageID, packageID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(basePath+"/actions/resize", body)
	if err != nil {
		return fmt.Errorf("failed to resize backup storage %s: %w", storageID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runDelete(cmd *cobra.Command, args []string) error {
	storageID, _ := cmd.Flags().GetString("storage-id")
	if err := requireStorageID(storageID); err != nil {
		return err
	}

	body := vdbclient.ActionBody(vdbclient.ResourceTypeBackupStorage, storageID, "delete", nil)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	printDeleteTarget(apiClient, storageID)

	if dryRun {
		vdbclient.PreviewBody("release", fmt.Sprintf("backup storage %s", storageID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Release backup storage %s? Backups that depend on it may be lost.", storageID)) {
		fmt.Println("Aborted.")
		return nil
	}

	// Singular /actions/delete here; the relational endpoint is /actions/deletions.
	result, err := apiClient.Post(basePath+"/actions/delete", body)
	if err != nil {
		return fmt.Errorf("failed to release backup storage %s: %w", storageID, err)
	}

	return vdbclient.Output(cmd, result)
}

func printDeleteTarget(apiClient *vdbclient.Client, storageID string) {
	result, err := apiClient.Get(basePath, nil)
	if err != nil {
		return
	}
	items, ok := vdbclient.Unwrap(result).([]interface{})
	if !ok {
		return
	}

	for _, item := range items {
		storage, ok := item.(map[string]interface{})
		if !ok || storage["id"] != storageID {
			continue
		}
		fmt.Println("The following backup storage will be released:")
		fmt.Println()
		fmt.Printf("  ID:      %v\n", storage["id"])
		fmt.Printf("  Name:    %v\n", storage["name"])
		fmt.Printf("  Package: %s\n", orDash(storage["backupPackageName"]))
		fmt.Printf("  Usage:   %v of %v\n", storage["usage"], storage["quota"])
		fmt.Println()
		return
	}
}

// orDash renders a value the API often leaves null.
func orDash(value interface{}) string {
	text, ok := value.(string)
	if !ok || text == "" {
		return "-"
	}
	return text
}
