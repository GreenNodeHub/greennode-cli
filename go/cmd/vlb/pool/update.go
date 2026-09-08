package pool

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a pool",
	Long: "Update a pool's algorithm, session settings or health monitor. The API " +
		"requires the full object, so unspecified fields are read from the current " +
		"pool and sent back unchanged.",
	RunE: runUpdate,
}

func init() {
	addLBFlag(updateCmd)
	addPoolFlag(updateCmd)
	f := updateCmd.Flags()
	f.String("algorithm", "", "Balancing algorithm: ROUND_ROBIN, LEAST_CONNECTIONS or SOURCE_IP")
	f.Bool("stickiness", false, "Enable sticky sessions")
	f.Bool("tls-encryption", false, "Enable TLS encryption to the members")
	f.String("health-monitor", "",
		"Health monitor fields to change, shorthand (timeout=5,interval=30) or JSON; merged into the current monitor")
	f.Bool("dry-run", false, "Print the request body without applying the update")
}

// updatableHealthMonitorFields are the UpdateHMonitorRequest fields carried over
// from the current monitor. healthCheckProtocol is deliberately absent: it is
// fixed at creation and rejected on update.
var updatableHealthMonitorFields = []string{
	"healthCheckPath", "healthCheckMethod", "healthyThreshold", "unhealthyThreshold",
	"timeout", "interval", "successCode", "domainName", "httpVersion",
}

// buildUpdateBody assembles the UpdatePoolRequest by seeding every required
// field from the pool's current state and overlaying only what the user set.
//
// `current` is the unwrapped PoolDto from GET; `currentMonitor` is the unwrapped
// HMonitorDto from the pool's healthMonitor endpoint. They are separate because
// GET /pools/{id} does NOT return the healthMonitor sub-object, despite PoolDto
// declaring it — it comes back with 11 scalar fields only. The update, however,
// demands a complete monitor (400 "If healthCheckProtocol field is HTTP, the
// following fields must be specified: ..."), so it has to be fetched from
// GET /pools/{id}/healthMonitor.
func buildUpdateBody(current, currentMonitor map[string]interface{}, changed map[string]bool,
	algorithm string, stickiness, tlsEncryption bool, healthMonitor string) (map[string]interface{}, error) {

	body := map[string]interface{}{}

	if changed["algorithm"] {
		body["algorithm"] = algorithm
	} else {
		body["algorithm"] = current["loadBalanceMethod"]
	}
	if body["algorithm"] == nil || body["algorithm"] == "" {
		return nil, fmt.Errorf("could not determine the pool's current algorithm; pass --algorithm explicitly")
	}

	// stickiness and tlsEncryption are gated on the pool's own protocol.
	// UpdatePoolValidator.validateUpdatePoolProperties re-runs the create-time
	// check (validateNonPoolConfiguration) against the protocol stored for the
	// pool, and that check tests != null — so echoing the service's own
	// "stickiness": false back at it on a TCP pool is rejected. GET /pools/{id}
	// reports both fields as false for every pool regardless of protocol, which
	// is exactly the trap: seeding from current state broke every non-HTTP
	// pool update. Same shape as the TLS fields on a non-HTTPS listener.
	protocol, _ := current["protocol"].(string)
	if protocol == poolProtocolHTTP {
		if changed["stickiness"] {
			body["stickiness"] = stickiness
		} else if v, ok := current["stickiness"]; ok {
			body["stickiness"] = v
		}
		if changed["tls-encryption"] {
			body["tlsEncryption"] = tlsEncryption
		} else if v, ok := current["tlsEncryption"]; ok {
			body["tlsEncryption"] = v
		}
	} else if err := rejectHTTPOnlyPoolFlags(
		httpOnlyFlagContext("this pool's protocol", protocol),
		changed["stickiness"], changed["tls-encryption"]); err != nil {
		return nil, err
	}

	hm := map[string]interface{}{}
	for _, k := range updatableHealthMonitorFields {
		v, ok := currentMonitor[k]
		if !ok || v == nil {
			continue
		}
		// An empty domainName is what the service reports when none is set;
		// echoing it back is pointless and risks tripping the httpVersion 1.1
		// domain check.
		if s, isStr := v.(string); isStr && s == "" {
			continue
		}
		hm[k] = v
	}
	overrides, err := cli.ParseStructFlagTyped(healthMonitor, healthMonitorIntFields, nil)
	if err != nil {
		return nil, fmt.Errorf("--health-monitor: %w", err)
	}
	for k, v := range overrides {
		hm[k] = v
	}
	if len(hm) == 0 {
		return nil, fmt.Errorf("could not read the pool's current health monitor; " +
			"pass the full monitor via --health-monitor")
	}
	// healthCheckProtocol is fixed at creation and stripped from the body, but
	// the backend still validates the body against it: UpdatePoolValidator reads
	// the stored protocol and re-runs validateNonPoolConfiguration, so an
	// override that adds an HTTP-shaped field to a TCP monitor is a 400.
	if protocol, ok := currentMonitor["healthCheckProtocol"].(string); ok {
		if err := validateMonitorFieldCombinations(protocol, hm); err != nil {
			return nil, err
		}
	}
	body["healthMonitor"] = hm

	return body, nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	lbID, poolID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}
	f := cmd.Flags()
	algorithm, _ := f.GetString("algorithm")
	stickiness, _ := f.GetBool("stickiness")
	tlsEncryption, _ := f.GetBool("tls-encryption")
	healthMonitor, _ := f.GetString("health-monitor")
	dryRun, _ := f.GetBool("dry-run")

	changed := map[string]bool{}
	for _, name := range []string{"algorithm", "stickiness", "tls-encryption", "health-monitor"} {
		changed[name] = f.Changed(name)
	}
	if !changed["algorithm"] && !changed["stickiness"] && !changed["tls-encryption"] && !changed["health-monitor"] {
		return fmt.Errorf("nothing to update: provide at least one of --algorithm, --stickiness, --tls-encryption, --health-monitor")
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	response, err := apiClient.Get(vlbclient.Pool(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch pool %s: %w", poolID, err)
	}
	// The pool payload omits its health monitor, so fetch that separately —
	// the update is rejected without a complete one.
	monitorResp, err := apiClient.Get(vlbclient.HealthMonitor(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch the health monitor of pool %s: %w", poolID, err)
	}

	body, err := buildUpdateBody(unwrapData(response), unwrapData(monitorResp), changed,
		algorithm, stickiness, tlsEncryption, healthMonitor)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("update", "pool "+poolID, body)
		return nil
	}

	result, err := apiClient.Put(vlbclient.Pool(projectID, lbID, poolID), body)
	if err != nil {
		return fmt.Errorf("failed to update pool %s: %w", poolID, err)
	}

	return outputRaw(cmd, cfg, result)
}
