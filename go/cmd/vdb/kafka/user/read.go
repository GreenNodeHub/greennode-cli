package user

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// userColumns is the table view of a UserDto. The four topic-name lists are left out
// on purpose: they are arrays, so a table cell cannot show them usefully and their
// presence would make OutputWithColumns pick one of them as the rows of a detail
// payload. The *All booleans stay, since they are what usually answers "may this
// user read everything?"; the lists come out of --output json or 'user get'.
var userColumns = []string{
	"id", "name", "status", "produceAll", "consumeAll", "produceConsumeAll",
	"adminAll", "mtlsAuthen", "saslAuthen", "createdAt",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the users of a Kafka cluster",
	Long: "List every user of a cluster with its authentication mechanisms and whether " +
		"its permissions cover all topics.\n\n" +
		"The per-topic lists are arrays and do not fit a table — use --output json, or " +
		"'user get' for one user.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a Kafka user",
	Long:  "Show one user of a cluster, including its per-topic permission lists.",
	Args:  cobra.NoArgs,
	RunE:  runGet,
}

var getCredsCmd = &cobra.Command{
	Use:   "get-creds",
	Short: "Show the authentication credentials of a Kafka user",
	Long: "Read back the credentials the cluster issued for a user.\n\n" +
		"THE OUTPUT IS SECRET: the API returns the credential material itself, so this " +
		"prints values that must not be pasted into a ticket or a shared log. Redirect " +
		"it to a file with restrictive permissions if you need to keep it.\n\n" +
		"Use 'user generate-creds' to replace them; that invalidates whatever this " +
		"returns now.",
	Args: cobra.NoArgs,
	RunE: runGetCreds,
}

func init() {
	listCmd.Flags().String("cluster-id", "", "Kafka cluster ID (required)")
	listCmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
	listCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck

	for _, cmd := range []*cobra.Command{getCmd, getCredsCmd} {
		f := cmd.Flags()
		f.String("cluster-id", "", "Kafka cluster ID (required)")
		f.String("user-id", "", "Kafka user ID (required)")
		cmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
		cmd.MarkFlagRequired("user-id")                                     //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("user-id", userIDCompletion())       //nolint:errcheck
	}
}

func runList(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := validateClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(usersPath(clusterID), nil)
	if err != nil {
		return fmt.Errorf("failed to list users of Kafka cluster %s: %w", clusterID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, userColumns)
}

func runGet(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	userID, _ := cmd.Flags().GetString("user-id")
	if err := validateIDs(clusterID, userID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(userPath(clusterID, userID), nil)
	if err != nil {
		return fmt.Errorf("failed to get user %s of Kafka cluster %s: %w", userID, clusterID, err)
	}

	// A user carries four nested arrays (the topic-name lists), so print key/value
	// rather than letting table extraction pick one of them at random.
	return vdbclient.Output(cmd, result)
}

func runGetCreds(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	userID, _ := cmd.Flags().GetString("user-id")
	if err := validateIDs(clusterID, userID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(userPath(clusterID, userID)+"/authen-creds", nil)
	if err != nil {
		return fmt.Errorf("failed to read the credentials of user %s of Kafka cluster %s: %w",
			userID, clusterID, err)
	}

	// The payload is a bare array of strings with no field names, so there is no
	// column set to apply; Output prints it as-is.
	return vdbclient.Output(cmd, result)
}
