package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new load balancer",
	RunE:  runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "Load balancer name (required)")
	f.String("package-id", "", "Package ID — run 'grn vlb package list' to see options (required)")
	f.String("scheme", "", "Scheme: Internet or Internal (required)")
	f.String("subnet-id", "", "Subnet ID the load balancer is placed in (required)")
	f.String("type", "", "Type: 'Layer 4' or 'Layer 7' (required)")
	f.String("zone-id", "", "Availability zone ID")
	f.Bool("auto-scalable", false, "Enable autoscale mode")
	f.StringArray("tag", nil, "Tag as key=value or JSON (repeatable)")
	f.String("listener", "", "Initial listener, shorthand (listenerName=web,listenerProtocol=HTTP,listenerProtocolPort=80) or JSON")
	f.String("pool", "", "Initial pool, shorthand (poolName=web,poolProtocol=HTTP,algorithm=ROUND_ROBIN) or JSON")
	f.Bool("dry-run", false, "Validate parameters and print the request body without creating")

	for _, name := range []string{"name", "package-id", "scheme", "subnet-id", "type"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

// listenerIntFields / listenerBoolFields drive shorthand type coercion for the
// nested --listener struct flag; the same key sets are reused by the listener
// package, so keep them in sync with CreateListenerRequest.
var (
	listenerIntFields = []string{"listenerProtocolPort", "timeoutClient", "timeoutMember", "timeoutConnection"}
	poolBoolFields    = []string{"stickiness", "tlsEncryption"}
)

// buildCreateBody assembles the CreateLoadBalancerRequest. Optional keys are
// omitted entirely rather than sent as null, so the backend applies its own
// defaults.
func buildCreateBody(name, packageID, scheme, subnetID, lbType, zoneID string, autoScalable bool,
	tags []string, listenerStr, poolStr string) (map[string]interface{}, error) {

	body := map[string]interface{}{
		"name":      name,
		"packageId": packageID,
		"scheme":    scheme,
		"subnetId":  subnetID,
		"type":      lbType,
	}
	if zoneID != "" {
		body["zoneId"] = zoneID
	}
	if autoScalable {
		body["autoScalable"] = true
	}

	parsedTags, err := cli.ParseStructFlags("tag", tags, nil, nil)
	if err != nil {
		return nil, err
	}
	if parsedTags != nil {
		body["tags"] = parsedTags
	}

	if listenerStr != "" {
		l, err := cli.ParseStructFlagTyped(listenerStr, listenerIntFields, nil)
		if err != nil {
			return nil, fmt.Errorf("--listener: %w", err)
		}
		body["listener"] = l
	}
	if poolStr != "" {
		p, err := cli.ParseStructFlagTyped(poolStr, nil, poolBoolFields)
		if err != nil {
			return nil, fmt.Errorf("--pool: %w", err)
		}
		body["pool"] = p
	}

	return body, nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()
	name, _ := f.GetString("name")
	packageID, _ := f.GetString("package-id")
	scheme, _ := f.GetString("scheme")
	subnetID, _ := f.GetString("subnet-id")
	lbType, _ := f.GetString("type")
	zoneID, _ := f.GetString("zone-id")
	autoScalable, _ := f.GetBool("auto-scalable")
	tags, _ := f.GetStringArray("tag")
	listenerStr, _ := f.GetString("listener")
	poolStr, _ := f.GetString("pool")
	dryRun, _ := f.GetBool("dry-run")

	body, err := buildCreateBody(name, packageID, scheme, subnetID, lbType, zoneID, autoScalable, tags, listenerStr, poolStr)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("create", "load balancer "+name, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(vlbclient.LoadBalancers(projectID), body)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	return outputRaw(cmd, cfg, result)
}
