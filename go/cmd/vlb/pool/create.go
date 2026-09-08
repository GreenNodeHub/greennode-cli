package pool

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

// memberIntFields / memberBoolFields drive shorthand type coercion for the
// --member struct flag; they mirror MemberRequest.
var (
	memberIntFields  = []string{"port", "weight", "monitorPort"}
	memberBoolFields = []string{"backup"}
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pool",
	RunE:  runCreate,
}

func init() {
	addLBFlag(createCmd)
	f := createCmd.Flags()
	f.String("name", "", "Pool name (required)")
	f.String("protocol", "", "Pool protocol: HTTP, TCP, UDP or PROXY (required)")
	f.String("algorithm", "ROUND_ROBIN", "Balancing algorithm: ROUND_ROBIN, LEAST_CONNECTIONS or SOURCE_IP")
	f.Bool("stickiness", false, "Enable sticky sessions")
	f.Bool("tls-encryption", false, "Enable TLS encryption to the members")
	f.String("health-monitor", "",
		"Health monitor, shorthand (healthCheckProtocol=HTTP,healthCheckPath=/,timeout=5,interval=30) or JSON (required)")
	f.StringArray("member", nil,
		"Pool member, shorthand (ipAddress=10.0.0.4,port=80,weight=1,backup=false) or JSON (repeatable)")
	f.Bool("dry-run", false, "Validate parameters and print the request body without creating")
	for _, name := range []string{"name", "protocol", "health-monitor"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

// healthMonitorAlwaysRequired are the fields the backend needs on every health
// monitor. The spec marks only healthCheckProtocol as required, but omitting
// any of these makes the service fail with HTTP 500 rather than a validation
// error — verified against the live API.
var healthMonitorAlwaysRequired = []string{"timeout", "interval", "healthyThreshold", "unhealthyThreshold"}

// healthMonitorHTTPRequired are additionally required when healthCheckProtocol
// is HTTP or HTTPS, per the backend's own 400 message.
var healthMonitorHTTPRequired = []string{"healthCheckPath", "healthCheckMethod", "successCode", "httpVersion"}

// validateHealthMonitor reports what the backend would reject, in one message
// the user can act on. These rules are not in the OpenAPI spec; they were found
// by exercising the live API, where the failures are either an HTTP 500 or a
// 400 naming a field the user never supplied.
func validateHealthMonitor(hm map[string]interface{}) error {
	var missing []string
	for _, f := range healthMonitorAlwaysRequired {
		if _, ok := hm[f]; !ok {
			missing = append(missing, f)
		}
	}
	protocol, _ := hm["healthCheckProtocol"].(string)
	if protocol == "" {
		missing = append(missing, "healthCheckProtocol")
	}
	if protocol == "HTTP" || protocol == "HTTPS" {
		for _, f := range healthMonitorHTTPRequired {
			if _, ok := hm[f]; !ok {
				missing = append(missing, f)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("--health-monitor is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if v, _ := hm["httpVersion"].(string); v == "1.1" {
		if _, ok := hm["domainName"]; !ok {
			return fmt.Errorf("--health-monitor: domainName is required when httpVersion is 1.1")
		}
	}
	return nil
}

// validateMembers reports members the backend would reject. monitorPort is
// optional in the spec but mandatory in practice: omitting it yields a 400
// complaining that monitorPort is out of range, which is baffling when the user
// never set one.
func validateMembers(members []map[string]interface{}) error {
	for i, m := range members {
		if _, ok := m["ipAddress"]; !ok {
			return fmt.Errorf("--member[%d]: ipAddress is required", i)
		}
		if _, ok := m["port"]; !ok {
			return fmt.Errorf("--member[%d]: port is required", i)
		}
		if _, ok := m["monitorPort"]; !ok {
			return fmt.Errorf("--member[%d]: monitorPort is required by the API "+
				"(set it to the same value as port for a plain health check)", i)
		}
	}
	return nil
}

// buildCreateBody assembles the CreatePoolRequest.
func buildCreateBody(name, protocol, algorithm string, stickiness, tlsEncryption bool,
	healthMonitor string, members []string) (map[string]interface{}, error) {

	hm, err := cli.ParseStructFlagTyped(healthMonitor, healthMonitorIntFields, nil)
	if err != nil {
		return nil, fmt.Errorf("--health-monitor: %w", err)
	}
	if hm == nil {
		return nil, fmt.Errorf("--health-monitor must not be empty")
	}

	parsedMembers, err := cli.ParseStructFlags("member", members, memberIntFields, memberBoolFields)
	if err != nil {
		return nil, err
	}

	// Every flag is parsed before any is validated, so a malformed flag is
	// reported as malformed rather than being masked by a semantic complaint
	// about a different one.
	if err := validateHealthMonitor(hm); err != nil {
		return nil, err
	}
	if err := validateMembers(parsedMembers); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"poolName":      name,
		"poolProtocol":  protocol,
		"algorithm":     algorithm,
		"stickiness":    stickiness,
		"tlsEncryption": tlsEncryption,
		"healthMonitor": hm,
	}
	if parsedMembers != nil {
		body["members"] = parsedMembers
	}
	return body, nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	lbID, _, err := idsOf(cmd, false)
	if err != nil {
		return err
	}
	f := cmd.Flags()
	name, _ := f.GetString("name")
	protocol, _ := f.GetString("protocol")
	algorithm, _ := f.GetString("algorithm")
	stickiness, _ := f.GetBool("stickiness")
	tlsEncryption, _ := f.GetBool("tls-encryption")
	healthMonitor, _ := f.GetString("health-monitor")
	members, _ := f.GetStringArray("member")
	dryRun, _ := f.GetBool("dry-run")

	body, err := buildCreateBody(name, protocol, algorithm, stickiness, tlsEncryption, healthMonitor, members)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("create", "pool "+name+" on load balancer "+lbID, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(vlbclient.Pools(projectID, lbID), body)
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	return outputRaw(cmd, cfg, result)
}
