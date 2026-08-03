package rule

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new rule in a security group",
	RunE:  runCreate,
}

func init() {
	f := createCmd.Flags()

	// Required
	f.String("secgroup-id", "", "Security group ID (required)")
	f.String("direction", "", "Traffic direction: ingress or egress (required)")
	f.String("protocol", "", "Protocol: tcp, udp, icmp, or any (required)")

	f.Int("port-range-min", 0, "Minimum port number (required for tcp/udp; not valid for icmp/any)")
	f.Int("port-range-max", 0, "Maximum port number (required for tcp/udp; not valid for icmp/any)")
	f.String("remote-ip-prefix", "", "Remote CIDR, e.g. 0.0.0.0/0 (required)")
	f.String("remote-group-id", "", "Remote security group ID (required)")
	f.String("ether-type", "IPv4", "Ether type: IPv4 or IPv6 (required)")

	for _, name := range []string{"secgroup-id", "direction", "protocol", "port-range-min", "port-range-max", "ether-type", "remote-ip-prefix"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
	f.String("description", "", "Rule description")
}

var validProtocols = map[string]bool{
	"tcp": true, "udp": true, "icmp": true, "any": true,
}

func runCreate(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	direction, _ := cmd.Flags().GetString("direction")
	protocol, _ := cmd.Flags().GetString("protocol")
	portMin, _ := cmd.Flags().GetInt("port-range-min")
	portMax, _ := cmd.Flags().GetInt("port-range-max")
	remoteIP, _ := cmd.Flags().GetString("remote-ip-prefix")
	remoteGroupID, _ := cmd.Flags().GetString("remote-group-id")
	etherType, _ := cmd.Flags().GetString("ether-type")
	description, _ := cmd.Flags().GetString("description")

	if err := validator.ValidateID(secgroupID, "secgroup-id"); err != nil {
		return err
	}

	if direction != "ingress" && direction != "egress" {
		return fmt.Errorf("--direction must be 'ingress' or 'egress', got %q", direction)
	}

	proto := strings.ToLower(protocol)
	if !validProtocols[proto] {
		return fmt.Errorf("--protocol must be one of tcp, udp, icmp, any — got %q", protocol)
	}

	portMinSet := cmd.Flags().Changed("port-range-min")
	portMaxSet := cmd.Flags().Changed("port-range-max")

	if (proto == "icmp" || proto == "any") && (portMinSet || portMaxSet) {
		return fmt.Errorf("--port-range-min/max must not be set when protocol is %q", protocol)
	}

	if portMinSet && portMaxSet && portMin > portMax {
		return fmt.Errorf("--port-range-min (%d) must be ≤ --port-range-max (%d)", portMin, portMax)
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"direction":      direction,
		"protocol":       proto,
		"etherType":      etherType,
		"remoteIpPrefix": nilIfEmpty(remoteIP),
		"remoteGroupId":  nilIfEmpty(remoteGroupID),
		"description":    nilIfEmpty(description),
	}

	// Only include port range fields when explicitly set by the user
	if portMinSet {
		body["portRangeMin"] = portMin
	}
	if portMaxSet {
		body["portRangeMax"] = portMax
	}

	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/secgroups/%s/secgroupRules", projectID, secgroupID), body)
	if err != nil {
		return fmt.Errorf("failed to create rule in security group %s: %w", secgroupID, err)
	}

	return outputResult(cmd, cfg, result)
}
