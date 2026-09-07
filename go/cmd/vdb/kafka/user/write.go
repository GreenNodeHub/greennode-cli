package user

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// permissions are the four permission kinds a Kafka user can hold. Each is a pair:
// a list of topic names and an "all topics" switch that, when true, makes the API
// IGNORE the list (its own words). They are declared once and looped over so a fifth
// kind cannot be half-added.
var permissions = []struct {
	listFlag  string // --produce-topics
	allFlag   string // --produce-all
	listField string // produceTopicNames
	allField  string // produceAll
	label     string
}{
	{"produce-topics", "produce-all", "produceTopicNames", "produceAll", "produce"},
	{"consume-topics", "consume-all", "consumeTopicNames", "consumeAll", "consume"},
	{"produce-consume-topics", "produce-consume-all", "produceConsumeTopicNames", "produceConsumeAll", "produce+consume"},
	{"admin-topics", "admin-all", "adminTopicNames", "adminAll", "admin"},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a user on a Kafka cluster",
	Long: "Create a user with a set of per-topic permissions.\n\n" +
		"Topics are named, not identified: --produce-topics and friends take topic " +
		"NAMES from 'grn vdb kafka topic list'.\n\n" +
		"Each --*-all switch OVERRIDES its list — the API ignores the topic names when " +
		"the matching switch is true, so passing both is not an error, just a list that " +
		"does nothing.\n\n" +
		"--mtls-authen and --sasl-authen say which mechanisms the user may use. They " +
		"have no effect unless the cluster accepts the same mechanism; check " +
		"'cluster get'. Read the resulting credentials with 'user get-creds'.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Change the permissions of a Kafka user",
	Long: "Replace a user's permissions and authentication mechanisms.\n\n" +
		"The endpoint takes the full set on every call, so flags left out keep their " +
		"current values: the CLI reads the user first and repeats what you did not " +
		"change. Without that, omitting a permission would revoke it.\n\n" +
		"A permission list you DO pass replaces the previous one — it is not merged. " +
		"Run 'user get' first to see what is there.",
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user from a Kafka cluster",
	Long: "Delete a user and its credentials.\n\n" +
		"Anything connecting as that user stops working immediately.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

var generateCredsCmd = &cobra.Command{
	Use:   "generate-creds",
	Short: "Reissue the authentication credentials of a Kafka user",
	Long: "Replace a user's credentials with new ones.\n\n" +
		"THIS INVALIDATES THE CURRENT CREDENTIALS: every client still using them is cut " +
		"off until it is given the new ones, which are read with 'user get-creds'.\n\n" +
		"The API returns no usable body here, so this command reports the outcome and " +
		"leaves fetching the new values to 'user get-creds' — that way the secret is " +
		"printed only when it is asked for.",
	Args: cobra.NoArgs,
	RunE: runGenerateCreds,
}

func init() {
	c := createCmd.Flags()
	c.String("cluster-id", "", "Kafka cluster ID (required)")
	c.String("name", "", "User name (required)")
	c.Bool("mtls-authen", false, "Let this user authenticate with mTLS")
	c.Bool("sasl-authen", false, "Let this user authenticate with SASL")
	createCmd.MarkFlagRequired("name") //nolint:errcheck

	u := updateCmd.Flags()
	u.String("cluster-id", "", "Kafka cluster ID (required)")
	u.String("user-id", "", "Kafka user ID (required)")
	u.Bool("mtls-authen", false, "Let this user authenticate with mTLS (default: unchanged)")
	u.Bool("sasl-authen", false, "Let this user authenticate with SASL (default: unchanged)")
	u.Bool("dry-run", false, "Print the request that would be sent without changing anything")
	u.Bool("force", false, "Skip the confirmation prompt")

	for _, cmd := range []*cobra.Command{createCmd, updateCmd} {
		for _, permission := range permissions {
			cmd.Flags().String(permission.listFlag, "",
				fmt.Sprintf("Topic NAMES this user may %s, comma-separated", permission.label))
			cmd.Flags().Bool(permission.allFlag, false,
				fmt.Sprintf("Let this user %s on ALL topics (overrides --%s)", permission.label, permission.listFlag))
			cmd.RegisterFlagCompletionFunc(permission.listFlag, topicNameCompletion()) //nolint:errcheck
		}
	}

	d := deleteCmd.Flags()
	d.String("cluster-id", "", "Kafka cluster ID (required)")
	d.String("user-id", "", "Kafka user ID (required)")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")

	g := generateCredsCmd.Flags()
	g.String("cluster-id", "", "Kafka cluster ID (required)")
	g.String("user-id", "", "Kafka user ID (required)")
	g.Bool("dry-run", false, "Print the request that would be sent without reissuing anything")
	g.Bool("force", false, "Skip the confirmation prompt")

	// Bound here, next to the flags — see the init-order note in completion.go.
	for _, cmd := range []*cobra.Command{createCmd, updateCmd, deleteCmd, generateCredsCmd} {
		cmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	}
	for _, cmd := range []*cobra.Command{updateCmd, deleteCmd, generateCredsCmd} {
		cmd.MarkFlagRequired("user-id")                               //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("user-id", userIDCompletion()) //nolint:errcheck
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := validateClusterID(clusterID); err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	mtls, _ := cmd.Flags().GetBool("mtls-authen")
	sasl, _ := cmd.Flags().GetBool("sasl-authen")

	body := map[string]interface{}{
		"name":       name,
		"mtlsAuthen": mtls,
		"saslAuthen": sasl,
	}
	for _, permission := range permissions {
		list, _ := cmd.Flags().GetString(permission.listFlag)
		all, _ := cmd.Flags().GetBool(permission.allFlag)
		body[permission.listField] = cli.ParseCommaSeparated(list)
		body[permission.allField] = all
	}

	if !mtls && !sasl {
		fmt.Println("Warning: neither --mtls-authen nor --sasl-authen was given, so this user has no way to authenticate.")
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(usersPath(clusterID), body)
	if err != nil {
		return fmt.Errorf("failed to create user %q on Kafka cluster %s: %w", name, clusterID, err)
	}

	// This POST returns the created user (UserDto), unwrapped.
	return vdbclient.Output(cmd, result)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	userID, _ := cmd.Flags().GetString("user-id")
	if err := validateIDs(clusterID, userID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	// The request replaces the whole permission set, so unspecified flags are filled
	// from the user as it stands — otherwise changing one permission would revoke the
	// other three.
	current, err := fetchUser(apiClient, clusterID, userID)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"mtlsAuthen": boolFlagOr(cmd, "mtls-authen", current["mtlsAuthen"]),
		"saslAuthen": boolFlagOr(cmd, "sasl-authen", current["saslAuthen"]),
	}
	for _, permission := range permissions {
		if cmd.Flags().Changed(permission.listFlag) {
			list, _ := cmd.Flags().GetString(permission.listFlag)
			body[permission.listField] = cli.ParseCommaSeparated(list)
		} else {
			body[permission.listField] = stringList(current[permission.listField])
		}
		body[permission.allField] = boolFlagOr(cmd, permission.allFlag, current[permission.allField])
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("update",
			fmt.Sprintf("user %s of Kafka cluster %s", userID, clusterID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Replace the permissions of user %s on Kafka cluster %s? Clients relying on a revoked permission stop working.",
		userID, clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	// The response is an unspecified bare string; the HTTP status is the result.
	if err := apiClient.NoContent("PUT", userPath(clusterID, userID), nil, body); err != nil {
		return fmt.Errorf("failed to update user %s of Kafka cluster %s: %w", userID, clusterID, err)
	}

	fmt.Printf("User %s of Kafka cluster %s updated. Run 'grn vdb kafka user get --cluster-id %s --user-id %s' to confirm.\n",
		userID, clusterID, clusterID, userID)
	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	userID, _ := cmd.Flags().GetString("user-id")
	if err := validateIDs(clusterID, userID); err != nil {
		return err
	}

	path := userPath(clusterID, userID)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would send DELETE %s\n", path)
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Delete user %s from Kafka cluster %s? Anything connecting as that user stops working.",
		userID, clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	if err := apiClient.NoContent("DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete user %s from Kafka cluster %s: %w", userID, clusterID, err)
	}

	fmt.Printf("User %s deleted from Kafka cluster %s.\n", userID, clusterID)
	return nil
}

func runGenerateCreds(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	userID, _ := cmd.Flags().GetString("user-id")
	if err := validateIDs(clusterID, userID); err != nil {
		return err
	}

	path := userPath(clusterID, userID) + "/regenerate-creds"

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would send PUT %s\n", path)
		cli.DryRunNotice("reissue the credentials")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Reissue the credentials of user %s on Kafka cluster %s? The current ones stop working immediately.",
		userID, clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	if err := apiClient.NoContent("PUT", path, nil, nil); err != nil {
		return fmt.Errorf("failed to reissue the credentials of user %s of Kafka cluster %s: %w",
			userID, clusterID, err)
	}

	fmt.Printf("New credentials issued for user %s of Kafka cluster %s. Read them with 'grn vdb kafka user get-creds --cluster-id %s --user-id %s' and give them to every client.\n",
		userID, clusterID, clusterID, userID)
	return nil
}

// boolFlagOr returns the flag when the user set it and the resource's current value
// otherwise. GetBool cannot tell "--flag=false" from "not passed"; Changed can.
func boolFlagOr(cmd *cobra.Command, name string, current interface{}) bool {
	if cmd.Flags().Changed(name) {
		value, _ := cmd.Flags().GetBool(name)
		return value
	}
	value, _ := current.(bool)
	return value
}

// stringList converts a decoded JSON array back into []string, dropping anything
// that is not a string. A nil becomes an empty slice so the field is sent as [] and
// not as null — the API distinguishes the two on a list it is going to replace.
func stringList(value interface{}) []string {
	items, _ := value.([]interface{})
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}
