package dhcp

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
)

// defaultDNSServers are always included when creating a DHCP option set.
var defaultDNSServers = []string{"10.166.12.196", "10.166.12.197"}

// maxAdditionalDNSServers is how many DNS servers may be added beyond the defaults.
const maxAdditionalDNSServers = 2

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new DHCP option set",
	Long: fmt.Sprintf(`Create a new DHCP option set.

The two default DNS servers (%s) are always included. You may add up to %d more
with --dns-server (so the set holds at most %d addresses in total).`,
		strings.Join(defaultDNSServers, ", "), maxAdditionalDNSServers, len(defaultDNSServers)+maxAdditionalDNSServers),
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "Name of the DHCP option set (required)")
	f.StringArray("dns-server", nil, fmt.Sprintf("Additional DNS server IP (repeatable, max %d)", maxAdditionalDNSServers))

	if err := createCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "name", err))
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	added, _ := cmd.Flags().GetStringArray("dns-server")

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}

	dnsServers, err := buildDNSServers(added)
	if err != nil {
		return err
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
		"name":       name,
		"dnsServers": dnsServers,
	}

	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/dhcp_option", projectID), body)
	if err != nil {
		return fmt.Errorf("failed to create DHCP option set: %w", err)
	}

	return outputResult(cmd, cfg, result)
}

// buildDNSServers prepends the fixed default DNS servers and appends the user-provided
// ones, validating each address and enforcing the additional-server limit. Defaults are
// never duplicated.
func buildDNSServers(added []string) ([]interface{}, error) {
	servers := make([]interface{}, 0, len(defaultDNSServers)+len(added))
	for _, ip := range defaultDNSServers {
		servers = append(servers, ip)
	}

	count := 0
	for _, raw := range added {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			continue
		}
		if net.ParseIP(ip) == nil {
			return nil, fmt.Errorf("invalid DNS server IP address: %q", raw)
		}
		// Skip a default that was passed again — it does not count toward the limit.
		if contains(defaultDNSServers, ip) {
			continue
		}
		count++
		if count > maxAdditionalDNSServers {
			return nil, fmt.Errorf("at most %d additional DNS server(s) may be added beyond the %d defaults", maxAdditionalDNSServers, len(defaultDNSServers))
		}
		servers = append(servers, ip)
	}
	return servers, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
