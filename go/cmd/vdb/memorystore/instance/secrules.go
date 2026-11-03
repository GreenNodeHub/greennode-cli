package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listSecrulesCmd = &cobra.Command{
	Use:   "list-secrules",
	Short: "List the security rules of a MemoryStore instance",
	Long: "List the security group rules controlling access to an instance.\n\n" +
		"Use 'instance update-secrule' to change them — note that update REPLACES the " +
		"whole rule set, so start from this listing.",
	Args: cobra.NoArgs,
	RunE: runListSecrules,
}

var updateSecruleCmd = &cobra.Command{
	Use:   "update-secrule",
	Short: "Replace the security rules of a MemoryStore instance",
	Long: "Set which networks may reach an instance.\n\n" +
		"The API REPLACES the entire rule set with what is sent, so by default the rules " +
		"you pass become the only rules — anything currently allowed and not repeated is " +
		"revoked. Pass --add to keep the existing rules and append instead.\n\n" +
		"Each --rule is a comma-separated list of key=value pairs:\n" +
		"  cidr      the allowed network, e.g. cidr=203.0.113.0/24 (required)\n" +
		"  port      single port; defaults to the instance's own port (6379)\n" +
		"  port-min  low end of a port range (use with port-max)\n" +
		"  port-max  high end of a port range\n" +
		"  id        existing rule ID, to keep a rule you are re-sending\n\n" +
		"Repeat --rule for each rule. Run 'instance list-secrules' first and --dry-run to " +
		"see exactly what would be sent; locking yourself out is the failure mode here.",
	Args: cobra.NoArgs,
	RunE: runUpdateSecrule,
}

func init() {
	l := listSecrulesCmd.Flags()
	l.String("instance-id", "", "MemoryStore instance ID (required)")
	listSecrulesCmd.MarkFlagRequired("instance-id")                                   //nolint:errcheck
	listSecrulesCmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck

	u := updateSecruleCmd.Flags()
	u.String("instance-id", "", "MemoryStore instance ID (required)")
	u.StringArray("rule", nil, "Rule as key=value pairs, repeatable (see --help)")
	u.Bool("add", false, "Append to the existing rules instead of replacing them")
	u.Bool("dry-run", false, "Print the request that would be sent without applying it")
	u.Bool("force", false, "Skip the confirmation prompt")
	updateSecruleCmd.MarkFlagRequired("instance-id")                                   //nolint:errcheck
	updateSecruleCmd.MarkFlagRequired("rule")                                          //nolint:errcheck
	updateSecruleCmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck
}

func runListSecrules(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := validateInstanceID(instanceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := fetchSecrules(apiClient, instanceID)
	if err != nil {
		return err
	}

	return vdbclient.OutputWithColumns(cmd, result, vdbclient.SecurityRuleColumns)
}

func runUpdateSecrule(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	instance, err := fetchInstance(apiClient, instanceID)
	if err != nil {
		return err
	}

	specs, _ := cmd.Flags().GetStringArray("rule")
	rules, err := vdbclient.ParseSecurityRules(specs, defaultPortOf(instance))
	if err != nil {
		return err
	}

	current, err := fetchSecrules(apiClient, instanceID)
	if err != nil {
		return err
	}
	existing := vdbclient.SecurityRulesFrom(current)

	add, _ := cmd.Flags().GetBool("add")
	if add {
		rules = append(existing, rules...)
	}

	fmt.Printf("Current rules on %s: %s\n", instanceID, vdbclient.DescribeSecurityRules(existing))
	fmt.Printf("Rules after this change: %s\n", vdbclient.DescribeSecurityRules(rules))
	if !add && len(existing) > 0 {
		fmt.Println("\nThis REPLACES the rule set; anything above that is not in the new set is revoked.")
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("update", fmt.Sprintf("the security rules of MemoryStore instance %s", instanceID), rules)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Apply this rule set to MemoryStore instance %s?", instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Put(instancePath(instanceID, "/secrules"), rules)
	if err != nil {
		return fmt.Errorf("failed to update the security rules of MemoryStore instance %s: %w", instanceID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, vdbclient.SecurityRuleColumns)
}

func fetchSecrules(apiClient *vdbclient.Client, instanceID string) (interface{}, error) {
	result, err := apiClient.Get(instancePath(instanceID, "/secrules"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list the security rules of MemoryStore instance %s: %w", instanceID, err)
	}
	return result, nil
}

// defaultPortOf is the port a rule gets when it names none: the instance's own port,
// falling back to Redis's 6379. There is only one engine here, so no per-engine map
// is needed — unlike relational, which serves MySQL (3306) and PostgreSQL (5432).
func defaultPortOf(instance map[string]interface{}) int {
	if port := intField(instance, "port"); port > 0 {
		return port
	}
	return redisPort
}
