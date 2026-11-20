package pool

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pool",
	RunE:  runCreate,
}

// exampleHealthMonitor is the --health-monitor example in the help text. A test
// checks that it passes validateHealthMonitor, so the help cannot drift again.
const exampleHealthMonitor = "healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3"

func init() {
	addLBFlag(createCmd)
	f := createCmd.Flags()
	f.String("name", "", "Pool name (required)")
	f.String("protocol", "", "Pool protocol: HTTP, TCP, UDP or PROXY (required)")
	f.String("algorithm", "ROUND_ROBIN", "Balancing algorithm: ROUND_ROBIN, LEAST_CONNECTIONS or SOURCE_IP")
	f.Bool("stickiness", false, "Enable sticky sessions")
	f.Bool("tls-encryption", false, "Enable TLS encryption to the members")
	f.String("health-monitor", "",
		"Health monitor, shorthand ("+exampleHealthMonitor+") or JSON (required); "+
			"HTTP also needs healthCheckPath, healthCheckMethod, successCode and httpVersion")
	f.StringArray("member", nil,
		"Pool member, shorthand (ipAddress=10.0.0.4,port=80,monitorPort=80,weight=1,backup=false) or JSON (repeatable)")
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

// healthMonitorHTTPOnly are the fields a TCP or PING-UDP monitor must not carry.
// CreatePoolValidator.validateHealthCheck rejects the whole request if any is
// present, and UpdatePoolValidator re-runs the same check against the monitor's
// stored protocol.
var healthMonitorHTTPOnly = []string{
	"healthCheckPath", "healthCheckMethod", "successCode", "domainName", "httpVersion",
}

// Health-check protocols that forbid every HTTP-shaped field.
var healthMonitorProtocolsWithoutHTTPFields = map[string]bool{"TCP": true, "PING-UDP": true}

var (
	healthCheckMethods = []string{"GET", "POST", "PUT"}
	healthCheckHTTPVer = []string{"1.0", "1.1"}
)

// validateHealthMonitor reports what the backend would reject, in one message
// the user can act on. These rules are not in the OpenAPI spec; they were found
// by exercising the live API, where the failures are either an HTTP 500 or a
// 400 naming a field the user never supplied.
func validateHealthMonitor(poolProtocol string, hm map[string]interface{}) error {
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
	switch protocol {
	case "HTTP":
		for _, f := range healthMonitorHTTPRequired {
			if _, ok := hm[f]; !ok {
				missing = append(missing, f)
			}
		}
	case "HTTPS":
		// Only the method is required for HTTPS: validateMethod rejects a nil
		// method, while validateHttpVersion returns early on nil and
		// validateHealthCheck demands path/successCode/httpVersion for HTTP
		// alone. Requiring all four here would have refused requests the API
		// accepts.
		if _, ok := hm["healthCheckMethod"]; !ok {
			missing = append(missing, "healthCheckMethod")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("--health-monitor is missing required field(s): %s", strings.Join(missing, ", "))
	}

	if err := validateHealthCheckPairing(poolProtocol, protocol); err != nil {
		return err
	}
	if err := validateMonitorTimings(hm); err != nil {
		return err
	}
	return validateMonitorFieldCombinations(protocol, hm)
}

// validateHealthCheckPairing enforces the UDP pairing rule. It holds regardless
// of the load balancer type: on a Layer 4 load balancer
// validateHealthCheckProtocol requires PING-UDP for a UDP pool and forbids it
// for any other, and on a Layer 7 one the pool is always HTTP and PING-UDP is
// not in the allowed set either way.
func validateHealthCheckPairing(poolProtocol, healthCheckProtocol string) error {
	switch {
	case poolProtocol == "UDP" && healthCheckProtocol != "PING-UDP":
		return fmt.Errorf("--health-monitor: healthCheckProtocol must be PING-UDP for a UDP pool, got %s",
			healthCheckProtocol)
	case poolProtocol != "" && poolProtocol != "UDP" && healthCheckProtocol == "PING-UDP":
		return fmt.Errorf("--health-monitor: healthCheckProtocol PING-UDP is only valid for a UDP pool, "+
			"but --protocol is %s", poolProtocol)
	}
	return nil
}

// validateMonitorTimings mirrors "Timeout must be less than interval". Only the
// create path checks it; the update path is left to the API so the CLI is never
// stricter than the service.
func validateMonitorTimings(hm map[string]interface{}) error {
	timeout, okT := monitorInt(hm["timeout"])
	interval, okI := monitorInt(hm["interval"])
	if okT && okI && timeout >= interval {
		return fmt.Errorf("--health-monitor: timeout must be less than interval, got timeout=%d interval=%d",
			timeout, interval)
	}
	return nil
}

// validateMonitorFieldCombinations covers the rules that depend only on the
// health-check protocol, so both create and update can apply them.
func validateMonitorFieldCombinations(healthCheckProtocol string, hm map[string]interface{}) error {
	if healthMonitorProtocolsWithoutHTTPFields[healthCheckProtocol] {
		var forbidden []string
		for _, f := range healthMonitorHTTPOnly {
			if _, ok := hm[f]; ok {
				forbidden = append(forbidden, f)
			}
		}
		if len(forbidden) > 0 {
			return fmt.Errorf("--health-monitor: %s cannot be set when healthCheckProtocol is %s",
				strings.Join(forbidden, ", "), healthCheckProtocol)
		}
		return nil
	}

	if healthCheckProtocol == "HTTP" || healthCheckProtocol == "HTTPS" {
		if m, ok := hm["healthCheckMethod"].(string); ok && !containsString(healthCheckMethods, m) {
			return fmt.Errorf("--health-monitor: healthCheckMethod must be one of %s, got %q",
				strings.Join(healthCheckMethods, ", "), m)
		}
	}

	v, hasVersion := hm["httpVersion"].(string)
	if hasVersion && !containsString(healthCheckHTTPVer, v) {
		return fmt.Errorf("--health-monitor: httpVersion must be one of %s, got %q",
			strings.Join(healthCheckHTTPVer, ", "), v)
	}
	_, hasDomain := hm["domainName"]
	switch {
	case v == "1.1" && !hasDomain:
		return fmt.Errorf("--health-monitor: domainName is required when httpVersion is 1.1")
	case v == "1.0" && hasDomain:
		return fmt.Errorf("--health-monitor: domainName cannot be set when httpVersion is 1.0")
	}
	return nil
}

// monitorInt reads a threshold or timing that may have arrived as an int from
// shorthand coercion or a float64 from JSON.
func monitorInt(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	}
	return 0, false
}

func containsString(all []string, want string) bool {
	for _, v := range all {
		if v == want {
			return true
		}
	}
	return false
}

// buildCreateBody assembles the CreatePoolRequest.
//
// stickiness and tlsEncryption are gated on the pool protocol. The backend tests
// them with != null (CreatePoolValidator.validateNonPoolConfiguration), so
// sending false is as fatal as sending true: an HTTP pool must carry both, and a
// non-HTTP pool must carry neither. Sending them unconditionally made every
// TCP/UDP/PROXY pool — that is, every pool on a Layer 4 load balancer —
// impossible to create.
func buildCreateBody(name, protocol, algorithm string, stickiness, tlsEncryption bool,
	stickinessSet, tlsEncryptionSet bool,
	healthMonitor string, members []string) (map[string]interface{}, error) {

	hm, err := cli.ParseStructFlagTyped(healthMonitor, healthMonitorIntFields, nil)
	if err != nil {
		return nil, fmt.Errorf("--health-monitor: %w", err)
	}
	if hm == nil {
		return nil, fmt.Errorf("--health-monitor must not be empty")
	}

	parsedMembers, err := cli.ParseStructFlags("member", members, vlbclient.MemberIntFields, vlbclient.MemberBoolFields)
	if err != nil {
		return nil, err
	}

	// Every flag is parsed before any is validated, so a malformed flag is
	// reported as malformed rather than being masked by a semantic complaint
	// about a different one.
	if err := validateHealthMonitor(protocol, hm); err != nil {
		return nil, err
	}
	if err := vlbclient.ValidateMembers(parsedMembers); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"poolName":      name,
		"poolProtocol":  protocol,
		"algorithm":     algorithm,
		"healthMonitor": hm,
	}
	if protocol == poolProtocolHTTP {
		body["stickiness"] = stickiness
		body["tlsEncryption"] = tlsEncryption
	} else {
		if err := rejectHTTPOnlyPoolFlags(
			httpOnlyFlagContext("--protocol", protocol), stickinessSet, tlsEncryptionSet); err != nil {
			return nil, err
		}
	}
	if parsedMembers != nil {
		body["members"] = parsedMembers
	}
	return body, nil
}

// poolProtocolHTTP is the only pool protocol that accepts — and requires —
// stickiness and tlsEncryption.
const poolProtocolHTTP = "HTTP"

// rejectHTTPOnlyPoolFlags fails when the user asked for a session- or TLS-related
// setting the backend forbids on this protocol, rather than dropping the flag
// silently or letting the API answer with a 400. protocolDesc names where the
// protocol came from, since create reads it from --protocol while update reads
// it back off the pool.
func rejectHTTPOnlyPoolFlags(protocolDesc string, stickinessSet, tlsEncryptionSet bool) error {
	var flags string
	switch {
	case stickinessSet && tlsEncryptionSet:
		flags = "--stickiness and --tls-encryption are"
	case stickinessSet:
		flags = "--stickiness is"
	case tlsEncryptionSet:
		flags = "--tls-encryption is"
	default:
		return nil
	}
	return fmt.Errorf("%s only valid for an HTTP pool, but %s", flags, protocolDesc)
}

// httpOnlyFlagContext renders the protocol for rejectHTTPOnlyPoolFlags, coping
// with a protocol the service did not report.
func httpOnlyFlagContext(source, protocol string) string {
	if protocol == "" {
		return source + " is not HTTP"
	}
	return source + " is " + protocol
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

	body, err := buildCreateBody(name, protocol, algorithm, stickiness, tlsEncryption,
		f.Changed("stickiness"), f.Changed("tls-encryption"), healthMonitor, members)
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
