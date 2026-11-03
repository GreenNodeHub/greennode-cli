package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// allowedPorts are the ports the API accepts on a rule, quoted from its own
// description of SecurityGroupRuleCreateRequest.port. Checked client-side so a typo
// names the flag rather than coming back as a 400.
var allowedPorts = []int{9092, 9094, 9096, 9194, 9196}

// Kafka's security rules are managed one at a time — POST to add, DELETE to remove —
// unlike relational and memorystore, where a single PUT replaces the whole rule set.
// Nothing from vdbclient's ParseSecurityRules/SecurityRulesFrom applies here.

var createSecruleCmd = &cobra.Command{
	Use:   "create-secrule",
	Short: "Allow a remote IP range to reach a Kafka cluster",
	Long: "Add one security rule: a remote IP prefix and the port it may reach.\n\n" +
		"Rules are added and removed individually here, one endpoint each — unlike " +
		"Relational Database and MemoryStore, where a single call replaces the whole " +
		"rule set. Existing rules are untouched.\n\n" +
		"Allowed ports are 9092, 9094, 9096, 9194 and 9196; which one applies depends on " +
		"the cluster's authentication and whether the client is inside the VPC.",
	Args: cobra.NoArgs,
	RunE: runCreateSecrule,
}

var deleteSecruleCmd = &cobra.Command{
	Use:   "delete-secrule",
	Short: "Remove a security rule from a Kafka cluster",
	Long: "Delete one security rule by ID.\n\n" +
		"Run 'cluster list-secrules' for the IDs. Clients covered only by this rule lose " +
		"access as soon as it is gone.",
	Args: cobra.NoArgs,
	RunE: runDeleteSecrule,
}

func init() {
	c := createSecruleCmd.Flags()
	c.String("cluster-id", "", "Kafka cluster ID (required)")
	c.String("remote-ip", "", "Remote IP prefix in CIDR form, e.g. 10.0.0.0/24 (required)")
	c.Int("port", 0, fmt.Sprintf("Port to allow, one of %v (required)", allowedPorts))
	c.Bool("dry-run", false, "Print the request that would be sent without creating the rule")
	c.Bool("force", false, "Skip the confirmation prompt")
	createSecruleCmd.MarkFlagRequired("remote-ip")                        //nolint:errcheck
	createSecruleCmd.MarkFlagRequired("port")                             //nolint:errcheck
	createSecruleCmd.RegisterFlagCompletionFunc("port", portCompletion()) //nolint:errcheck

	d := deleteSecruleCmd.Flags()
	d.String("cluster-id", "", "Kafka cluster ID (required)")
	d.String("secrule-id", "", "ID of the rule to delete (required; 'cluster list-secrules')")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")
	deleteSecruleCmd.MarkFlagRequired("secrule-id")                                  //nolint:errcheck
	deleteSecruleCmd.RegisterFlagCompletionFunc("secrule-id", secruleIDCompletion()) //nolint:errcheck

	for _, cmd := range []*cobra.Command{createSecruleCmd, deleteSecruleCmd} {
		cmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	}
}

func portCompletion() cli.CompFunc {
	values := make([]string, 0, len(allowedPorts))
	for _, port := range allowedPorts {
		values = append(values, fmt.Sprint(port))
	}
	return cli.FlagValues(values...)
}

func runCreateSecrule(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	remoteIP, _ := cmd.Flags().GetString("remote-ip")
	port, _ := cmd.Flags().GetInt("port")
	if !isAllowedPort(port) {
		return fmt.Errorf("invalid --port %d: a Kafka security rule accepts only %v", port, allowedPorts)
	}

	body := map[string]interface{}{
		"remoteIp": remoteIP,
		"port":     port,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("create", fmt.Sprintf("a security rule on Kafka cluster %s", clusterID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Allow %s to reach Kafka cluster %s on port %d?", remoteIP, clusterID, port)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(ClusterPath(clusterID, "/security-group-rules"), body)
	if err != nil {
		return fmt.Errorf("failed to create a security rule on Kafka cluster %s: %w", clusterID, err)
	}

	// This POST does return the created rule (SecurityGroupRuleDto), unwrapped —
	// unlike its DELETE counterpart, whose response is a bare string.
	return vdbclient.Output(cmd, result)
}

func runDeleteSecrule(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	secruleID, _ := cmd.Flags().GetString("secrule-id")
	if err := validator.ValidateID(secruleID, "secrule-id"); err != nil {
		return err
	}

	path := ClusterPath(clusterID, "/security-group-rules/"+secruleID)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would send DELETE %s\n", path)
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Delete security rule %s from Kafka cluster %s? Clients covered only by it lose access.",
		secruleID, clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	if err := apiClient.NoContent("DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete security rule %s from Kafka cluster %s: %w",
			secruleID, clusterID, err)
	}

	fmt.Printf("Security rule %s deleted from Kafka cluster %s.\n", secruleID, clusterID)
	return nil
}

func isAllowedPort(port int) bool {
	for _, allowed := range allowedPorts {
		if port == allowed {
			return true
		}
	}
	return false
}
