package vks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

func TestValidateAddonName(t *testing.T) {
	for _, ok := range []string{"block-storage-csi", "coredns", "a1"} {
		if err := validateAddonName(ok); err != nil {
			t.Errorf("validateAddonName(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "a", "Coredns", "-coredns", "coredns-", "core_dns", "../x", strings.Repeat("a", 64)} {
		if err := validateAddonName(bad); err == nil {
			t.Errorf("validateAddonName(%q) = nil, want error", bad)
		}
	}
}

func TestValidateAddonVersion(t *testing.T) {
	for _, ok := range []string{"1.2.0-vks.1", "v1.13.1-vks.12", "1.2.0-rc.1-vks.2"} {
		if err := validateAddonVersion(ok); err != nil {
			t.Errorf("validateAddonVersion(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "1.2.0", "latest", "1.2.0-vks.1/../x", "1.2-vks.1"} {
		if err := validateAddonVersion(bad); err == nil {
			t.Errorf("validateAddonVersion(%q) = nil, want error", bad)
		}
	}
}

func TestNormalizeResolveConflicts(t *testing.T) {
	if got, err := normalizeResolveConflicts("overwrite"); err != nil || got != "OVERWRITE" {
		t.Errorf("normalizeResolveConflicts(overwrite) = %q, %v", got, err)
	}
	if _, err := normalizeResolveConflicts("merge"); err == nil {
		t.Error("normalizeResolveConflicts(merge) = nil error, want error")
	}
}

func TestReadConfigurationValuesFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.yaml")
	if err := os.WriteFile(path, []byte("controller:\n  replicaCount: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readConfigurationValues("file://" + path)
	if err != nil || got != "controller:\n  replicaCount: 2\n" {
		t.Errorf("readConfigurationValues(file://) = %q, %v", got, err)
	}
	if got, _ := readConfigurationValues(`{"a":1}`); got != `{"a":1}` {
		t.Errorf("inline value = %q, want verbatim", got)
	}
	if _, err := readConfigurationValues("file://" + filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("missing file: want error")
	}
	if _, err := readConfigurationValues(strings.Repeat("x", maxConfigurationValuesLength+1)); err == nil {
		t.Error("oversized value: want error")
	}
}

// newAddonFlagCmd returns a command with the create/update-addon request flags,
// parsed from args.
func newAddonFlagCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "x"}
	addAddonRequestFlags(c, "", "")
	if err := c.Flags().Parse(args); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBuildCreateAddonBodyOnlySetFields(t *testing.T) {
	c := newAddonFlagCmd(t, "--resolve-conflicts", "overwrite")
	a, changed, err := readAddonFlags(c)
	if err != nil {
		t.Fatal(err)
	}
	got := buildCreateAddonBody("coredns", "kube-system", a, changed)
	if got["addonName"] != "coredns" || got["resolveConflicts"] != "OVERWRITE" {
		t.Errorf("body = %#v", got)
	}
	ns, _ := got["namespaceConfig"].(map[string]interface{})
	if ns["namespace"] != "kube-system" {
		t.Errorf("namespaceConfig = %#v", got["namespaceConfig"])
	}
	for _, absent := range []string{"addonVersion", "configurationValues", "clientRequestToken"} {
		if _, ok := got[absent]; ok {
			t.Errorf("%s should be absent when its flag is not set; body = %#v", absent, got)
		}
	}
	if _, ok := buildCreateAddonBody("coredns", "", a, changed)["namespaceConfig"]; ok {
		t.Error("namespaceConfig should be absent without --namespace")
	}
}

func TestBuildAddonBodyKeepsEmptyConfigurationValues(t *testing.T) {
	c := newAddonFlagCmd(t, "--configuration-values", "")
	a, changed, err := readAddonFlags(c)
	if err != nil {
		t.Fatal(err)
	}
	got := buildAddonBody(a, changed)
	if v, ok := got["configurationValues"]; !ok || v != "" {
		t.Errorf(`--configuration-values "" must be sent as "" (clears values); body = %#v`, got)
	}
	if len(got) != 1 {
		t.Errorf("body has %d keys, want 1: %#v", len(got), got)
	}
}

func TestReadAddonFlagsRejectsBadValues(t *testing.T) {
	for _, args := range [][]string{
		{"--addon-version", "latest"},
		{"--resolve-conflicts", "merge"},
		{"--client-request-token", strings.Repeat("t", 65)},
	} {
		if _, _, err := readAddonFlags(newAddonFlagCmd(t, args...)); err == nil {
			t.Errorf("readAddonFlags(%v) = nil error, want error", args)
		}
	}
}

func TestBuildAddonVersionsQuery(t *testing.T) {
	got := buildAddonVersionsQuery("coredns", "v1.32.9", "vks, community", "", "", true)
	want := map[string]string{
		"addonName": "coredns", "kubernetesVersion": "1.32.9", "owners": "vks,community", "includeDeprecated": "true",
	}
	if len(got) != len(want) {
		t.Fatalf("query = %#v, want %#v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("query[%s] = %q, want %q", k, got[k], v)
		}
	}
	if q := buildAddonVersionsQuery("", "", "", "", "", false); len(q) != 0 {
		t.Errorf("empty flags: query = %#v, want empty", q)
	}
}

func addonResp(status string) map[string]interface{} {
	return map[string]interface{}{"addon": map[string]interface{}{"status": status}}
}

func TestEvaluateAddonActive(t *testing.T) {
	notFound := &client.APIError{StatusCode: 404}
	forbidden := &client.APIError{StatusCode: 403}
	cases := []struct {
		name                          string
		result                        interface{}
		err                           error
		wantDone, wantFail, wantFatal bool
	}{
		{"active", addonResp("ACTIVE"), nil, true, false, false},
		{"creating", addonResp("CREATING"), nil, false, false, false},
		{"degraded keeps polling", addonResp("DEGRADED"), nil, false, false, false},
		{"updating", addonResp("UPDATING"), nil, false, false, false},
		{"create failed", addonResp("CREATE_FAILED"), nil, false, true, false},
		{"update failed", addonResp("UPDATE_FAILED"), nil, false, true, false},
		{"top-level status ignored", map[string]interface{}{"status": "ACTIVE"}, nil, false, false, false},
		{"transient err", nil, errors.New("boom"), false, false, false},
		{"404 fatal", nil, notFound, false, false, true},
		{"403 fatal", nil, forbidden, false, false, true},
	}
	for _, tc := range cases {
		done, failed, _, fatal := evaluateAddonActive(tc.result, tc.err)
		if done != tc.wantDone || failed != tc.wantFail || (fatal != nil) != tc.wantFatal {
			t.Errorf("%s: done=%v failed=%v fatal=%v, want %v %v %v",
				tc.name, done, failed, fatal != nil, tc.wantDone, tc.wantFail, tc.wantFatal)
		}
	}
}

func TestEvaluateAddonDeleted(t *testing.T) {
	notFound := &client.APIError{StatusCode: 404}
	forbidden := &client.APIError{StatusCode: 403}
	cases := []struct {
		name                          string
		result                        interface{}
		err                           error
		wantDone, wantFail, wantFatal bool
	}{
		{"gone", nil, notFound, true, false, false},
		{"deleting", addonResp("DELETING"), nil, false, false, false},
		{"delete failed", addonResp("DELETE_FAILED"), nil, false, true, false},
		{"403 fatal", nil, forbidden, false, false, true},
		{"transient", nil, errors.New("boom"), false, false, false},
	}
	for _, tc := range cases {
		done, failed, _, fatal := evaluateAddonDeleted(tc.result, tc.err)
		if done != tc.wantDone || failed != tc.wantFail || (fatal != nil) != tc.wantFatal {
			t.Errorf("%s: done=%v failed=%v fatal=%v, want %v %v %v",
				tc.name, done, failed, fatal != nil, tc.wantDone, tc.wantFail, tc.wantFatal)
		}
	}
}

func TestEvaluateUpdateSuccessful(t *testing.T) {
	upd := func(s string) map[string]interface{} {
		return map[string]interface{}{"update": map[string]interface{}{"status": s}}
	}
	notFound := &client.APIError{StatusCode: 404}
	cases := []struct {
		name                          string
		result                        interface{}
		err                           error
		wantDone, wantFail, wantFatal bool
	}{
		{"successful", upd("Successful"), nil, true, false, false},
		{"in progress", upd("InProgress"), nil, false, false, false},
		{"failed", upd("Failed"), nil, false, true, false},
		{"cancelled", upd("Cancelled"), nil, false, true, false},
		{"404 fatal", nil, notFound, false, false, true},
	}
	for _, tc := range cases {
		done, failed, _, fatal := evaluateUpdateSuccessful(tc.result, tc.err)
		if done != tc.wantDone || failed != tc.wantFail || (fatal != nil) != tc.wantFatal {
			t.Errorf("%s: done=%v failed=%v fatal=%v, want %v %v %v",
				tc.name, done, failed, fatal != nil, tc.wantDone, tc.wantFail, tc.wantFatal)
		}
	}
}

func TestExtractStrings(t *testing.T) {
	resp := map[string]interface{}{"addons": []interface{}{"coredns", "", 3, "block-storage-csi"}}
	got := extractStrings(resp, "addons")
	if len(got) != 2 || got[0] != "coredns" || got[1] != "block-storage-csi" {
		t.Errorf("extractStrings = %#v", got)
	}
	if extractStrings("nope", "addons") != nil {
		t.Error("non-map response: want nil")
	}
}

// TestAddonFlagCompletionsBound asserts every add-on related flag has a value
// completer, so completion cannot silently be left unwired.
func TestAddonFlagCompletionsBound(t *testing.T) {
	cmds := []*cobra.Command{
		listAddonsCmd, getAddonCmd, createAddonCmd, updateAddonCmd, deleteAddonCmd,
		listAddonVersionsCmd, getAddonConfigurationSchemaCmd, listUpdatesCmd, getUpdateCmd,
		waitAddonActiveSub, waitAddonDeletedSub, waitUpdateSuccessfulSub,
	}
	flags := []string{"addon-name", "addon-version", "resolve-conflicts", "update-id", "cluster-id"}
	for _, c := range cmds {
		for _, name := range flags {
			if c.Flags().Lookup(name) == nil {
				continue
			}
			// wait subcommands are not walked by registerCompletions (see CLAUDE.md).
			if name == "cluster-id" && c.Parent() == waitCmd {
				continue
			}
			if _, ok := c.GetFlagCompletionFunc(name); !ok {
				t.Errorf("%s --%s has no completion function", c.CommandPath(), name)
			}
		}
	}
}
