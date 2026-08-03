package server

import (
	"fmt"
	"regexp"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var serverNameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_]{0,63}[a-zA-Z0-9]$`)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new vServer instance",
	RunE:  runCreate,
}

func init() {
	f := createCmd.Flags()

	// Required
	f.String("name", "", "Server name (required)")
	f.String("flavor-id", "", "Flavor ID — run 'vserver flavor list' to see options (required)")
	f.String("image-id", "", "Image ID — run 'vserver image list --type os|gpu' to see options (required)")
	f.String("network-id", "", "VPC ID — run 'vserver vpc list' to see options (required)")
	f.String("subnet-id", "", "Subnet ID — run 'vserver subnet list --vpc-id <id>' to see options (required)")
	f.String("root-disk-type-id", "", "Volume type ID — run 'vserver volume-type list' to see options (required)")
	f.String("zone-id", "", "Availability zone ID — run without this flag to see available zones (required)")

	if err := createCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "name", err))
	}

	// Root disk
	f.Int("root-disk-size", 20, "Root disk size in GiB (minimum 20)")
	f.String("root-disk-encryption-type", "", "Root disk encryption type")
	f.Bool("encryption-volume", false, "Encrypt the root volume")

	// Data disk (optional)
	f.String("data-disk-type-id", "", "Data disk volume type ID")
	f.Int("data-disk-size", 0, "Data disk size in GiB (0 = no data disk)")
	f.String("data-disk-encryption-type", "", "Data disk encryption type")
	f.String("data-disk-name", "", "Data disk name")

	// Network
	f.Bool("attach-floating", false, "Attach a floating IP to the server")
	f.String("security-group", "", "Security group IDs (comma-separated)")

	// Auth
	f.String("ssh-key-id", "", "SSH key ID to inject into the server")
	f.String("user-name", "", "OS login username")
	f.String("user-password", "", "OS login password")
	f.Bool("expire-password", true, "Force password change on first login")

	// Placement
	f.String("server-group-id", "", "Server group ID for placement policy")
	f.String("host-group-id", "", "Dedicated host group ID")

	// Backup / restore
	f.Bool("enable-backup", false, "Enable backup for the server")
	f.String("backup-instance-point-id", "", "Backup instance point ID to restore from")
	f.String("snapshot-instance-point-id", "", "Snapshot instance point ID to restore from")

	// Billing
	f.Int("period", 1, "Billing period in months")
	f.Bool("is-poc", false, "Mark as PoC (proof-of-concept) instance")
	f.Bool("is-enable-auto-renew", false, "Enable auto-renewal")
	f.Bool("os-licence", false, "Include OS licence in billing")

	// User data
	f.String("user-data", "", "User data script passed to cloud-init")
	f.Bool("user-data-base64-encoded", false, "Indicate that --user-data value is already base64-encoded")

	f.Bool("dry-run", false, "Validate parameters without creating the server")
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	flavorID, _ := cmd.Flags().GetString("flavor-id")
	imageID, _ := cmd.Flags().GetString("image-id")
	networkID, _ := cmd.Flags().GetString("network-id")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	rootDiskTypeID, _ := cmd.Flags().GetString("root-disk-type-id")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rootDiskSize, _ := cmd.Flags().GetInt("root-disk-size")
	rootDiskEncType, _ := cmd.Flags().GetString("root-disk-encryption-type")
	encryptionVolume, _ := cmd.Flags().GetBool("encryption-volume")
	dataDiskTypeID, _ := cmd.Flags().GetString("data-disk-type-id")
	dataDiskSize, _ := cmd.Flags().GetInt("data-disk-size")
	dataDiskEncType, _ := cmd.Flags().GetString("data-disk-encryption-type")
	dataDiskName, _ := cmd.Flags().GetString("data-disk-name")
	attachFloating, _ := cmd.Flags().GetBool("attach-floating")
	securityGroup, _ := cmd.Flags().GetString("security-group")
	sshKeyID, _ := cmd.Flags().GetString("ssh-key-id")
	userName, _ := cmd.Flags().GetString("user-name")
	userPassword, _ := cmd.Flags().GetString("user-password")
	expirePassword, _ := cmd.Flags().GetBool("expire-password")
	serverGroupID, _ := cmd.Flags().GetString("server-group-id")
	hostGroupID, _ := cmd.Flags().GetString("host-group-id")
	enableBackup, _ := cmd.Flags().GetBool("enable-backup")
	backupPointID, _ := cmd.Flags().GetString("backup-instance-point-id")
	snapshotPointID, _ := cmd.Flags().GetString("snapshot-instance-point-id")
	period, _ := cmd.Flags().GetInt("period")
	isPoc, _ := cmd.Flags().GetBool("is-poc")
	autoRenew, _ := cmd.Flags().GetBool("is-enable-auto-renew")
	osLicence, _ := cmd.Flags().GetBool("os-licence")
	userData, _ := cmd.Flags().GetString("user-data")
	userDataB64, _ := cmd.Flags().GetBool("user-data-base64-encoded")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dryRun {
		return validateCreate(name, flavorID, imageID, networkID, subnetID, rootDiskTypeID, zoneID, rootDiskSize)
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	if zoneID == "" {
		return suggestZones(apiClient, projectID)
	}
	if networkID == "" {
		return suggestVPCs(apiClient, projectID)
	}
	if subnetID == "" {
		return suggestSubnets(apiClient, projectID, networkID)
	}
	if imageID == "" {
		return suggestImages()
	}
	if flavorID == "" {
		return suggestFlavors()
	}
	if rootDiskTypeID == "" {
		return suggestRootDiskTypes(zoneID)
	}

	body := map[string]interface{}{
		"isPoc":                   isPoc,
		"name":                    name,
		"flavorId":                flavorID,
		"imageId":                 imageID,
		"networkId":               networkID,
		"subnetId":                subnetID,
		"rootDiskTypeId":          rootDiskTypeID,
		"rootDiskSize":            rootDiskSize,
		"rootDiskEncryptionType":  nilIfEmpty(rootDiskEncType),
		"encryptionVolume":        encryptionVolume,
		"attachFloating":          attachFloating,
		"securityGroup":           parseCommaSeparated(securityGroup),
		"sshKeyId":                nilIfEmpty(sshKeyID),
		"serverGroupId":           nilIfEmpty(serverGroupID),
		"hostGroupId":             nilIfEmpty(hostGroupID),
		"expirePassword":          expirePassword,
		"osLicence":               osLicence,
		"enableBackup":            enableBackup,
		"backupInstancePointId":   nilIfEmpty(backupPointID),
		"snapshotInstancePointId": nilIfEmpty(snapshotPointID),
		"createdFrom":             "NEW",
		"tags":                    []interface{}{},
		"configVolumeRestores":    []interface{}{},
		"userData":                nilIfEmpty(userData),
		"userDataBase64Encoded":   userDataB64,
		"zoneId":                  zoneID,
		"dataDiskTypeId":          nilIfEmpty(dataDiskTypeID),
		"dataDiskEncryptionType":  nilIfEmpty(dataDiskEncType),
		"dataDiskName":            nilIfEmpty(dataDiskName),
		"period":                  period,
		"isEnableAutoRenew":       autoRenew,
	}

	if dataDiskSize > 0 {
		body["dataDiskSize"] = dataDiskSize
	} else {
		body["dataDiskSize"] = nil
	}

	if userName != "" {
		body["userName"] = userName
	}
	if userPassword != "" {
		body["userPassword"] = userPassword
	}

	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/servers", projectID), body)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return outputResult(cmd, cfg, transformServerResult(result))
}

func validateCreate(name, flavorID, imageID, networkID, subnetID, rootDiskTypeID, zoneID string, rootDiskSize int) error {
	var errs []string

	if len(name) < 5 || !serverNameRE.MatchString(name) {
		errs = append(errs, fmt.Sprintf(
			"server name %q is invalid — must be 5–65 chars, alphanumeric/hyphens/underscores, start/end with alphanumeric", name))
	}
	if rootDiskSize < 20 {
		errs = append(errs, fmt.Sprintf("root disk size %d GiB is too small (minimum 20 GiB)", rootDiskSize))
	}

	for _, check := range []struct{ val, flag string }{
		{flavorID, "flavor-id"},
		{imageID, "image-id"},
		{networkID, "network-id"},
		{subnetID, "subnet-id"},
		{rootDiskTypeID, "root-disk-type-id"},
	} {
		if err := validator.ValidateID(check.val, check.flag); err != nil {
			errs = append(errs, err.Error())
		}
	}

	fmt.Println("=== DRY RUN: Validation results ===")
	fmt.Println()
	if len(errs) > 0 {
		fmt.Printf("Found %d error(s):\n", len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("dry-run validation failed with %d error(s)", len(errs))
	}

	fmt.Println("All parameters are valid. Run without --dry-run to create the server.")
	return nil
}
