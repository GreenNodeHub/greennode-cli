package vks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

// Add-on API: /v1/clusters/{id}/addons, /v1/addon-versions and
// /v1/clusters/{id}/updates (vks-api, EKS-style add-on model).

// addonNamePattern mirrors the server-side add-on name rule (AddonConst.NAME_PATTERN).
var addonNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`)

// addonVersionPattern mirrors the server-side version rule (AddonConst.VERSION_PATTERN),
// e.g. 1.2.0-vks.1 or v1.13.1-rc.1-vks.2.
var addonVersionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?-vks\.[0-9]+$`)

// namespacePattern mirrors the Kubernetes namespace (RFC 1123 label) rule.
var namespacePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// resolveConflictsValues are the accepted --resolve-conflicts values.
var resolveConflictsValues = []string{"NONE", "OVERWRITE", "PRESERVE"}

// maxConfigurationValuesLength is the server-side limit of configurationValues (64 KiB).
const maxConfigurationValuesLength = 65536

func validateAddonName(name string) error {
	if !addonNamePattern.MatchString(name) {
		return fmt.Errorf("invalid addon-name: %q. Must be 2-63 lowercase alphanumeric characters or hyphens", name)
	}
	return nil
}

func validateAddonVersion(version string) error {
	if !addonVersionPattern.MatchString(version) {
		return fmt.Errorf("invalid addon-version: %q. Expected a VKS build such as 1.2.0-vks.1", version)
	}
	return nil
}

func validateNamespace(ns string) error {
	if len(ns) > 63 || !namespacePattern.MatchString(ns) {
		return fmt.Errorf("invalid namespace: %q. Must be a valid Kubernetes namespace name", ns)
	}
	return nil
}

// normalizeResolveConflicts upper-cases the value and checks it is one of NONE,
// OVERWRITE or PRESERVE.
func normalizeResolveConflicts(v string) (string, error) {
	u := strings.ToUpper(strings.TrimSpace(v))
	for _, ok := range resolveConflictsValues {
		if u == ok {
			return u, nil
		}
	}
	return "", fmt.Errorf("--resolve-conflicts must be one of NONE, OVERWRITE, PRESERVE, got %q", v)
}

// readConfigurationValues returns the --configuration-values value. A value of
// the form file://<path> is read from that file (JSON or YAML, sent verbatim).
func readConfigurationValues(v string) (string, error) {
	if path, ok := strings.CutPrefix(v, "file://"); ok {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("--configuration-values: %w", err)
		}
		v = string(data)
	}
	if len(v) > maxConfigurationValuesLength {
		return "", fmt.Errorf("--configuration-values must be at most 64 KiB, got %d bytes", len(v))
	}
	return v, nil
}

// addonFlags are the request fields shared by create-addon and update-addon.
type addonFlags struct {
	version            string
	configurationValue string
	resolveConflicts   string
	clientRequestToken string
}

// readAddonFlags reads and validates the shared flags. changed reports, per flag
// name, whether the user set it; only set flags go into the request body, so an
// update keeps whatever the user did not mention.
func readAddonFlags(cmd *cobra.Command) (addonFlags, map[string]bool, error) {
	f := cmd.Flags()
	var a addonFlags
	changed := map[string]bool{}
	for _, name := range []string{"addon-version", "configuration-values", "resolve-conflicts", "client-request-token"} {
		changed[name] = f.Changed(name)
	}
	a.version, _ = f.GetString("addon-version")
	a.resolveConflicts, _ = f.GetString("resolve-conflicts")
	a.clientRequestToken, _ = f.GetString("client-request-token")
	raw, _ := f.GetString("configuration-values")

	if changed["addon-version"] {
		if err := validateAddonVersion(a.version); err != nil {
			return a, changed, err
		}
	}
	if changed["configuration-values"] {
		v, err := readConfigurationValues(raw)
		if err != nil {
			return a, changed, err
		}
		a.configurationValue = v
	}
	if changed["resolve-conflicts"] {
		v, err := normalizeResolveConflicts(a.resolveConflicts)
		if err != nil {
			return a, changed, err
		}
		a.resolveConflicts = v
	}
	if changed["client-request-token"] {
		if n := len(a.clientRequestToken); n < 1 || n > 64 {
			return a, changed, fmt.Errorf("--client-request-token must be 1 to 64 characters")
		}
	}
	return a, changed, nil
}

// buildAddonBody builds the CreateAddon/UpdateAddon body from the set flags.
// An explicitly empty --configuration-values ("") is kept: on update it clears
// the current values.
func buildAddonBody(a addonFlags, changed map[string]bool) map[string]interface{} {
	body := map[string]interface{}{}
	if changed["addon-version"] {
		body["addonVersion"] = a.version
	}
	if changed["configuration-values"] {
		body["configurationValues"] = a.configurationValue
	}
	if changed["resolve-conflicts"] {
		body["resolveConflicts"] = a.resolveConflicts
	}
	if changed["client-request-token"] {
		body["clientRequestToken"] = a.clientRequestToken
	}
	return body
}

// addAddonRequestFlags declares the flags shared by create-addon and update-addon.
func addAddonRequestFlags(cmd *cobra.Command, versionHelp, valuesHelp string) {
	f := cmd.Flags()
	f.String("addon-version", "", versionHelp)
	f.String("configuration-values", "", valuesHelp)
	f.String("resolve-conflicts", "", "How to treat fields changed outside VKS: NONE, OVERWRITE or PRESERVE")
	f.String("client-request-token", "", "Idempotency key, unique per cluster (1-64 characters)")
	_ = cmd.RegisterFlagCompletionFunc("resolve-conflicts", cli.FlagValues(resolveConflictsValues...))
	_ = cmd.RegisterFlagCompletionFunc("addon-version", cli.FlagFromAPI(fetchAddonVersions))
}

// nestedStatus returns obj[key].status for a decoded {key: {status: ...}} response.
func nestedStatus(result interface{}, key string) string {
	m, ok := result.(map[string]interface{})
	if !ok {
		return ""
	}
	return statusOf(m[key])
}

// evaluateAddonActive decides one poll for `wait addon-active`: ACTIVE succeeds;
// CREATE_FAILED and UPDATE_FAILED fail. Unlike the EKS AddonActive waiter,
// DEGRADED keeps polling: VKS reports it whenever the Helm release is applied
// but the workloads are not ready yet (e.g. a cluster with no nodes), and moves
// to ACTIVE on its own once they are.
func evaluateAddonActive(result interface{}, err error) (done bool, failed bool, status string, fatal error) {
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return false, false, "", err
		}
		if isPermanentAPIError(err) {
			return false, false, "", err
		}
		return false, false, "", nil
	}
	status = nestedStatus(result, "addon")
	switch status {
	case "ACTIVE":
		return true, false, status, nil
	case "CREATE_FAILED", "UPDATE_FAILED":
		return false, true, status, nil
	default:
		return false, false, status, nil
	}
}

// evaluateAddonDeleted decides one poll for `wait addon-deleted`: a 404 succeeds,
// DELETE_FAILED fails.
func evaluateAddonDeleted(result interface{}, err error) (done bool, failed bool, status string, fatal error) {
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return true, false, "DELETED", nil
		}
		if isPermanentAPIError(err) {
			return false, false, "", err
		}
		return false, false, "", nil
	}
	status = nestedStatus(result, "addon")
	if status == "DELETE_FAILED" {
		return false, true, status, nil
	}
	return false, false, status, nil
}

// evaluateUpdateSuccessful decides one poll for `wait update-successful`:
// Successful succeeds; Failed and Cancelled fail; a 404 is fatal.
func evaluateUpdateSuccessful(result interface{}, err error) (done bool, failed bool, status string, fatal error) {
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return false, false, "", err
		}
		if isPermanentAPIError(err) {
			return false, false, "", err
		}
		return false, false, "", nil
	}
	status = nestedStatus(result, "update")
	switch status {
	case "Successful":
		return true, false, status, nil
	case "Failed", "Cancelled":
		return false, true, status, nil
	default:
		return false, false, status, nil
	}
}

// extractStrings returns resp[key] when it is a list of strings.
func extractStrings(resp interface{}, key string) []string {
	m, ok := resp.(map[string]interface{})
	if !ok {
		return nil
	}
	items, _ := m[key].([]interface{})
	var out []string
	for _, it := range items {
		if s, ok := it.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// fetchClusterAddonNames completes --addon-name with the add-ons installed on
// --cluster-id.
func fetchClusterAddonNames(_ context.Context, cmd *cobra.Command) ([]string, error) {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if clusterID == "" {
		return nil, nil
	}
	c, err := createClient(cmd)
	if err != nil {
		return nil, err
	}
	res, err := c.Get(fmt.Sprintf("/v1/clusters/%s/addons", clusterID), nil)
	if err != nil {
		return nil, err
	}
	return extractStrings(res, "addons"), nil
}

// fetchCatalogAddonNames completes --addon-name with the add-ons of the catalog.
func fetchCatalogAddonNames(_ context.Context, cmd *cobra.Command) ([]string, error) {
	c, err := createClient(cmd)
	if err != nil {
		return nil, err
	}
	res, err := c.Get("/v1/addon-versions", nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(res, "addonName"), nil
}

// fetchAddonVersions completes --addon-version with the catalog versions of
// --addon-name (empty until --addon-name is set).
func fetchAddonVersions(_ context.Context, cmd *cobra.Command) ([]string, error) {
	name, _ := cmd.Flags().GetString("addon-name")
	if name == "" {
		return nil, nil
	}
	c, err := createClient(cmd)
	if err != nil {
		return nil, err
	}
	res, err := c.Get("/v1/addon-versions", map[string]string{"addonName": name})
	if err != nil {
		return nil, err
	}
	var out []string
	for _, addon := range asList(res, "addons") {
		m, _ := addon.(map[string]interface{})
		for _, v := range asList(m, "addonVersions") {
			if vm, ok := v.(map[string]interface{}); ok {
				if s, ok := vm["addonVersion"].(string); ok && s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out, nil
}

// fetchUpdateIDs completes --update-id with the latest updates of --cluster-id,
// scoped by --addon-name / --nodegroup-id when set.
func fetchUpdateIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if clusterID == "" {
		return nil, nil
	}
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	addonName, _ := cmd.Flags().GetString("addon-name")
	c, err := createClient(cmd)
	if err != nil {
		return nil, err
	}
	res, err := c.Get(fmt.Sprintf("/v1/clusters/%s/updates", clusterID), updateScopeParams(nodegroupID, addonName))
	if err != nil {
		return nil, err
	}
	return extractStrings(res, "updateIds"), nil
}

func asList(resp interface{}, key string) []interface{} {
	m, ok := resp.(map[string]interface{})
	if !ok {
		return nil
	}
	l, _ := m[key].([]interface{})
	return l
}
