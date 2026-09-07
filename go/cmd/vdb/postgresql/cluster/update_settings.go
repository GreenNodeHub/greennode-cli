package cluster

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateSettingsCmd = &cobra.Command{
	Use:   "update-settings",
	Short: "Update the master password or public access of a PostgreSQL Cluster",
	Long: "Change a cluster's master password, its public-access setting, or both.\n\n" +
		"Only the settings you pass are sent: --public-access is read solely when you " +
		"give it, so updating the password cannot silently flip network exposure.\n\n" +
		"The new password is read from $" + passwordEnv + " when --password is omitted, and " +
		"is masked in --dry-run output. Both changes affect live connections, so the " +
		"command confirms first.",
	Args: cobra.NoArgs,
	RunE: runUpdateSettings,
}

// settingsFlagsOn defines the flags on f; see the note on createFlags.
func settingsFlagsOn(f *pflag.FlagSet) {
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.String("password", "", "New master password (defaults to $"+passwordEnv+")")
	f.Bool("public-access", false, "Whether the cluster is reachable publicly")
	f.Bool("dry-run", false, "Print the request that would be sent without applying it")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	settingsFlagsOn(updateSettingsCmd.Flags())

	updateSettingsCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	updateSettingsCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
}

func runUpdateSettings(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	body, summary, err := settingsBody(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("update", fmt.Sprintf("the settings of PostgreSQL Cluster %s", clusterID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Update %s on PostgreSQL Cluster %s?", summary, clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(pgPath(clusterID, "/settings"), body)
	if err != nil {
		return fmt.Errorf("failed to update settings of PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}

// settingsBody assembles UpdatePostgreClusterSettingsRequest from the flags the
// user actually set, and describes the change for the prompt.
//
// Sending only changed fields matters here: publicAccess is a boolean whose zero
// value is a real, destructive setting. A body that always carried
// "publicAccess": false would cut off public connectivity every time someone
// rotated a password.
func settingsBody(cmd *cobra.Command) (map[string]interface{}, string, error) {
	flags := cmd.Flags()
	body := map[string]interface{}{}
	var changes []string

	password, _ := flags.GetString("password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}
	if password != "" {
		body["password"] = password
		changes = append(changes, "the master password")
	}

	if flags.Changed("public-access") {
		publicAccess, _ := flags.GetBool("public-access")
		body["publicAccess"] = publicAccess
		changes = append(changes, fmt.Sprintf("public access (-> %t)", publicAccess))
	}

	if len(body) == 0 {
		return nil, "", fmt.Errorf("nothing to update: pass --password (or set $%s) and/or --public-access", passwordEnv)
	}

	summary := changes[0]
	if len(changes) > 1 {
		summary = changes[0] + " and " + changes[1]
	}
	return body, summary, nil
}
