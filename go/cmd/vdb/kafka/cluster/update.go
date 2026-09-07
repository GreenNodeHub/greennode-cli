package cluster

import (
	"fmt"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

// All three commands in this file share a shape that no other vDB group has: the
// arguments go in the query string of a PUT with no body, and the response is an
// unspecified bare string, so the command reports the HTTP outcome and points at the
// read command that shows the effect.

var updateAuthenticationCmd = &cobra.Command{
	Use:   "update-authentication",
	Short: "Turn mTLS and SASL authentication on or off for a Kafka cluster",
	Long: "Set which authentication mechanisms the cluster accepts.\n\n" +
		"BOTH switches are sent on every call — the endpoint takes mtlsAuthen and " +
		"saslAuthen together, so this replaces the pair rather than editing one of them. " +
		"Pass both flags to be explicit; unset flags default to the cluster's current " +
		"values, which are read first.\n\n" +
		"THIS CAN BREAK EVERY CLIENT: turning a mechanism off invalidates the " +
		"credentials that use it, and turning both off leaves the cluster open to " +
		"anyone who can reach its network.",
	Args: cobra.NoArgs,
	RunE: runUpdateAuthentication,
}

var updateConfigGroupCmd = &cobra.Command{
	Use:   "update-config-group",
	Short: "Apply a config group version to a Kafka cluster",
	Long: "Point a cluster at a config group VERSION.\n\n" +
		"Kafka config groups are versioned and a cluster attaches to one specific " +
		"version, not to the group — so the value here is a version id from " +
		"'grn vdb kafka configuration get', not a group id. Editing a group means " +
		"creating a new version with 'configuration create-version' and then applying it " +
		"here.\n\n" +
		"Broker properties change under the cluster, so this can disrupt connected " +
		"clients.",
	Args: cobra.NoArgs,
	RunE: runUpdateConfigGroup,
}

var updatePublicAccessCmd = &cobra.Command{
	Use:   "update-public-access",
	Short: "Turn public access on or off for a Kafka cluster",
	Long: "Attach or detach the cluster's public (floating) addresses.\n\n" +
		"THIS CHANGES THE ADDRESSES CLIENTS CONNECT TO: turning it off drops the public " +
		"endpoints, and anything reaching the cluster from outside the VPC stops " +
		"working. Turning it on exposes the brokers to the internet, filtered only by " +
		"the cluster's security rules — check 'cluster list-secrules' first.",
	Args: cobra.NoArgs,
	RunE: runUpdatePublicAccess,
}

func init() {
	a := updateAuthenticationCmd.Flags()
	a.String("cluster-id", "", "Kafka cluster ID (required)")
	a.Bool("mtls-authen", false, "Accept mTLS authentication (default: leave as it is)")
	a.Bool("sasl-authen", false, "Accept SASL authentication (default: leave as it is)")
	a.Bool("dry-run", false, "Print the request that would be sent without changing anything")
	a.Bool("force", false, "Skip the confirmation prompt")

	c := updateConfigGroupCmd.Flags()
	c.String("cluster-id", "", "Kafka cluster ID (required)")
	c.String("config-group-version-id", "", "Config group VERSION to apply (required)")
	c.Bool("dry-run", false, "Print the request that would be sent without changing anything")
	c.Bool("force", false, "Skip the confirmation prompt")
	updateConfigGroupCmd.MarkFlagRequired("config-group-version-id")                                           //nolint:errcheck
	updateConfigGroupCmd.RegisterFlagCompletionFunc("config-group-version-id", configGroupVersionCompletion()) //nolint:errcheck

	p := updatePublicAccessCmd.Flags()
	p.String("cluster-id", "", "Kafka cluster ID (required)")
	p.Bool("enable", false, "Attach public addresses; omit or pass --enable=false to detach them")
	p.Bool("dry-run", false, "Print the request that would be sent without changing anything")
	p.Bool("force", false, "Skip the confirmation prompt")

	for _, cmd := range []*cobra.Command{updateAuthenticationCmd, updateConfigGroupCmd, updatePublicAccessCmd} {
		cmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	}
}

func runUpdateAuthentication(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// The endpoint requires both switches on every call, so an unset flag has to
	// mean "keep what the cluster has" rather than "false" — otherwise changing one
	// would silently turn the other off.
	current, err := fetchCluster(apiClient, clusterID)
	if err != nil {
		return err
	}
	mtls := boolFlagOr(cmd, "mtls-authen", current["mtlsAuthen"])
	sasl := boolFlagOr(cmd, "sasl-authen", current["saslAuthen"])

	params := map[string]string{
		"mtlsAuthen": strconv.FormatBool(mtls),
		"saslAuthen": strconv.FormatBool(sasl),
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewQuery("update", fmt.Sprintf("authentication of Kafka cluster %s", clusterID),
			"PUT", ClusterPath(clusterID, "/authentication"), params)
		return nil
	}

	prompt := fmt.Sprintf("Set Kafka cluster %s to mTLS=%t, SASL=%t? Clients using a disabled mechanism stop working.",
		clusterID, mtls, sasl)
	if !mtls && !sasl {
		prompt = fmt.Sprintf(
			"Turn OFF both mTLS and SASL on Kafka cluster %s? The brokers then accept unauthenticated clients.",
			clusterID)
	}
	if !cli.Confirm(force, prompt) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := apiClient.NoContent("PUT", ClusterPath(clusterID, "/authentication"), params, nil); err != nil {
		return fmt.Errorf("failed to update authentication of Kafka cluster %s: %w", clusterID, err)
	}

	fmt.Printf("Authentication change accepted for Kafka cluster %s (mTLS=%t, SASL=%t). Run 'grn vdb kafka cluster get --cluster-id %s' to confirm it applied.\n",
		clusterID, mtls, sasl, clusterID)
	return nil
}

func runUpdateConfigGroup(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	versionID, _ := cmd.Flags().GetString("config-group-version-id")
	params := map[string]string{"configGroupVersionId": versionID}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewQuery("update", fmt.Sprintf("config group of Kafka cluster %s", clusterID),
			"PUT", ClusterPath(clusterID, "/config-group"), params)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Apply config group version %s to Kafka cluster %s? Broker properties change under connected clients.",
		versionID, clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	if err := apiClient.NoContent("PUT", ClusterPath(clusterID, "/config-group"), params, nil); err != nil {
		return fmt.Errorf("failed to apply config group version %s to Kafka cluster %s: %w",
			versionID, clusterID, err)
	}

	fmt.Printf("Config group version %s accepted for Kafka cluster %s. Run 'grn vdb kafka cluster get --cluster-id %s' to confirm it applied.\n",
		versionID, clusterID, clusterID)
	return nil
}

func runUpdatePublicAccess(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	enable, _ := cmd.Flags().GetBool("enable")
	// The query parameter is declared as a STRING, not a boolean, unlike the
	// mtlsAuthen/saslAuthen pair on the sibling endpoint. Send the same "true"/"false"
	// text either way; do not assume the two endpoints agree just because both take a
	// switch.
	params := map[string]string{"enable": strconv.FormatBool(enable)}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewQuery("update", fmt.Sprintf("public access of Kafka cluster %s", clusterID),
			"PUT", ClusterPath(clusterID, "/public-access"), params)
		return nil
	}

	prompt := fmt.Sprintf(
		"Turn public access ON for Kafka cluster %s? The brokers become reachable from the internet, filtered only by its security rules.",
		clusterID)
	if !enable {
		prompt = fmt.Sprintf(
			"Turn public access OFF for Kafka cluster %s? Clients connecting to its public addresses stop working.",
			clusterID)
	}
	if !cli.Confirm(force, prompt) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	if err := apiClient.NoContent("PUT", ClusterPath(clusterID, "/public-access"), params, nil); err != nil {
		return fmt.Errorf("failed to update public access of Kafka cluster %s: %w", clusterID, err)
	}

	fmt.Printf("Public access change accepted for Kafka cluster %s (enable=%t). Run 'grn vdb kafka cluster get --cluster-id %s' to see the resulting addresses.\n",
		clusterID, enable, clusterID)
	return nil
}

// boolFlagOr returns the flag when the user set it, and the cluster's current value
// otherwise. Cobra's GetBool cannot distinguish "--flag=false" from "not passed", so
// Changed is the only way to tell them apart.
func boolFlagOr(cmd *cobra.Command, name string, current interface{}) bool {
	if cmd.Flags().Changed(name) {
		value, _ := cmd.Flags().GetBool(name)
		return value
	}
	value, _ := current.(bool)
	return value
}
