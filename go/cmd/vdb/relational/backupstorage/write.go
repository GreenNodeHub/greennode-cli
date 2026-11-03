package backupstorage

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Buy backup storage",
	Long: "Buy a backup storage package.\n\n" +
		"THIS COSTS MONEY: it goes through the order/payment flow and completes " +
		"asynchronously. Run 'backup-storage list-packages' for the package IDs and " +
		"prices, and 'backup-storage list' to see whether you already own storage — " +
		"growing what you have is 'backup-storage resize'.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Change the package of existing backup storage",
	Long: "Move existing backup storage to a different package, usually a larger one.\n\n" +
		"THIS COSTS MONEY: order/payment flow, asynchronous. Shrinking below what your " +
		"backups already occupy is rejected by the API — check the usage column in " +
		"'backup-storage list' first.",
	Args: cobra.NoArgs,
	RunE: runResize,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Release backup storage",
	Long: "Give up a backup storage quota.\n\n" +
		"Any backup that relies on this storage is at risk: release it only when your " +
		"backups fit in the free allowance, or after deleting them. Check " +
		"'backup-storage list' for usage and 'grn vdb relational backup list' for what " +
		"is stored.",
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
	r.String("package-id", "", "New package (required; see 'backup-storage list-packages')")
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

func runCreate(cmd *cobra.Command, args []string) error {
	packageID, _ := cmd.Flags().GetString("package-id")

	// CreateBackupStorageRequest is a single field — the only vdb create that is not
	// a description of the resource.
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

	// Same envelope as an instance resize, but the resource type is
	// dbaas-backup-storage — and the field is spelled resourceType, not resType.
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

	// The delete action uses resType (not resourceType) with the backup-storage
	// resource type. Note the endpoint is POST /actions/deletions, plural.
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

	result, err := apiClient.Post(basePath+"/actions/deletions", body)
	if err != nil {
		return fmt.Errorf("failed to release backup storage %s: %w", storageID, err)
	}

	return vdbclient.Output(cmd, result)
}

// printDeleteTarget shows the quota and, importantly, how much of it is in use.
func printDeleteTarget(apiClient *vdbclient.Client, storageID string) {
	result, err := apiClient.Get(basePath+"/information", nil)
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
		// backupPackageName comes back null even on a storage bought minutes ago
		// (verified live), so print a dash rather than "<nil>".
		fmt.Printf("  Package: %s\n", orDash(storage["backupPackageName"]))
		fmt.Printf("  Usage:   %v of %v\n", storage["usage"], storage["quota"])
		fmt.Println()
		return
	}
}

// orDash renders a value that the API often leaves null.
func orDash(value interface{}) string {
	text, ok := value.(string)
	if !ok || text == "" {
		return "-"
	}
	return text
}
