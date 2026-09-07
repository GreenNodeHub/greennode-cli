package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// postgresPort is the port a PostgreSQL Cluster listens on, and the only value
// the API documents for this product, so a rule that omits a port gets it.
const postgresPort = 5432

var updateSecruleCmd = &cobra.Command{
	Use:   "update-secrule",
	Short: "Replace the security rules of a PostgreSQL Cluster",
	Long: "Set which networks may reach a cluster.\n\n" +
		"The API REPLACES the entire rule set with what is sent, so by default the rules " +
		"you pass become the only rules — anything currently allowed and not repeated is " +
		"revoked. Pass --add to keep the existing rules and append instead.\n\n" +
		"Each --rule is a comma-separated list of key=value pairs:\n" +
		"  cidr      the allowed network, e.g. cidr=203.0.113.0/24 (required)\n" +
		"  port      single port; defaults to 5432\n" +
		"  port-min  low end of a port range (use with port-max)\n" +
		"  port-max  high end of a port range\n" +
		"  id        existing rule ID, to keep a rule you are re-sending\n\n" +
		"Repeat --rule for each rule. Run 'cluster list-secrules' first and --dry-run to " +
		"see exactly what would be sent; locking yourself out is the failure mode here.",
	Args: cobra.NoArgs,
	RunE: runUpdateSecrule,
}

func init() {
	f := updateSecruleCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.StringArray("rule", nil, "Rule as key=value pairs, repeatable (see --help)")
	f.Bool("add", false, "Append to the existing rules instead of replacing them")
	f.Bool("dry-run", false, "Print the request that would be sent without applying it")
	f.Bool("force", false, "Skip the confirmation prompt")

	updateSecruleCmd.MarkFlagRequired("cluster-id") //nolint:errcheck
	updateSecruleCmd.MarkFlagRequired("rule")       //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	updateSecruleCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
}

func runUpdateSecrule(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	specs, _ := cmd.Flags().GetStringArray("rule")
	rules, err := vdbclient.ParseSecurityRules(specs, postgresPort)
	if err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	add, _ := cmd.Flags().GetBool("add")
	existing, err := currentRules(apiClient, clusterID)
	if err != nil {
		return err
	}
	if add {
		rules = append(existing, rules...)
	}

	fmt.Printf("Current rules on %s: %s\n", clusterID, vdbclient.DescribeSecurityRules(existing))
	fmt.Printf("Rules after this change: %s\n", vdbclient.DescribeSecurityRules(rules))
	if !add && len(existing) > 0 {
		fmt.Println("\nThis REPLACES the rule set; anything above that is not in the new set is revoked.")
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		// The body is a bare JSON ARRAY, not an object — one of the six vdb
		// endpoints shaped that way.
		vdbclient.PreviewBody("update", fmt.Sprintf("the security rules of PostgreSQL Cluster %s", clusterID), rules)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Apply this rule set to PostgreSQL Cluster %s?", clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Put(relPath(clusterID, "/secrules"), rules)
	if err != nil {
		return fmt.Errorf("failed to update the security rules of PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, vdbclient.SecurityRuleColumns)
}

// currentRules reads the cluster's rules in the shape the update endpoint expects.
func currentRules(apiClient interface {
	Get(string, map[string]string) (interface{}, error)
}, clusterID string) ([]interface{}, error) {
	result, err := fetchSecrules(apiClient, clusterID)
	if err != nil {
		return nil, err
	}
	return vdbclient.SecurityRulesFrom(result), nil
}
