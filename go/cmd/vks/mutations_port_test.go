package vks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// https://docs.api.greennode.ai/service-docs/vks-api.html
var mutationPortCases = []struct {
	command      *cobra.Command
	method, path string
	status       int
	body         map[string]any
	args         []string
}{
	{acknowledgeKubeconfigWarningCmd, "PUT", "/v1/clusters/fixture-cluster/kubeconfig/acknowledge-warning", 204, nil, []string{"--cluster-id", "fixture-cluster"}},
	{createWorkspaceCmd, "POST", "/v1/workspace", 202, nil, nil},
	{registerFleetCmd, "POST", "/v1/clusters/fixture-cluster/register-fleet", 200, map[string]any{"fleetType": "NEW", "name": "fixture-fleet", "enableEastWestTraffic": false}, []string{"--cluster-id", "fixture-cluster", "--fleet-type", "NEW", "--fleet-name", "fixture-fleet", "--enable-east-west-traffic", "disabled"}},
	{resetWorkspaceServiceAccountCmd, "POST", "/v1/workspace/reset-service-account", 202, nil, nil},
	{stopPOCCmd, "POST", "/v1/clusters/fixture-cluster/stop-poc", 200, nil, []string{"--cluster-id", "fixture-cluster"}},
	{unregisterFleetCmd, "PUT", "/v1/clusters/fixture-cluster/unregister-fleet", 200, nil, []string{"--cluster-id", "fixture-cluster"}},
}

func mutationPortRoot(t *testing.T, original *cobra.Command) *cobra.Command {
	t.Helper()
	command := &cobra.Command{Use: original.Use, RunE: original.RunE}
	original.Flags().VisitAll(func(f *pflag.Flag) {
		switch f.Value.Type() {
		case "string":
			command.Flags().String(f.Name, f.DefValue, f.Usage)
		case "bool":
			command.Flags().Bool(f.Name, f.DefValue == "true", f.Usage)
		default:
			t.Fatalf("unexpected fixture flag type %s", f.Value.Type())
		}
		if len(f.Annotations[cobra.BashCompOneRequiredFlag]) > 0 {
			if err := command.MarkFlagRequired(f.Name); err != nil {
				t.Fatal(err)
			}
		}
	})
	return testutil.ProfileRoot(command)
}

func TestPortedMutationsDryRunWithoutProfile(t *testing.T) {
	for _, c := range mutationPortCases {
		t.Run(c.command.Name(), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			for _, name := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_PROFILE", "GRN_DEFAULT_PROJECT_ID", "GRN_PORTAL_USER_ID"} {
				t.Setenv(name, "")
			}
			cli.SetNonInteractive(true)
			t.Cleanup(func() { cli.SetNonInteractive(false) })
			root := mutationPortRoot(t, c.command)
			args := append([]string{c.command.Name()}, c.args...)
			root.SetArgs(append(args, "--dry-run"))
			out := testutil.CaptureStdout(t, func() {
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
			})
			if !strings.Contains(out, "=== DRY RUN ===") {
				t.Fatal("missing preview")
			}
			entries, err := os.ReadDir(home)
			if err != nil || len(entries) != 0 {
				t.Fatal("dry-run wrote profile files")
			}
		})
	}
}

func TestPortedDestructiveMutationsRequireConfirmation(t *testing.T) {
	for _, c := range mutationPortCases {
		if c.command.Flags().Lookup("force") == nil {
			continue
		}
		t.Run(c.command.Name(), func(t *testing.T) {
			cli.SetNonInteractive(true)
			t.Cleanup(func() { cli.SetNonInteractive(false) })
			root := mutationPortRoot(t, c.command)
			root.SetArgs(append([]string{c.command.Name()}, c.args...))
			testutil.CaptureStdout(t, func() {
				err := root.Execute()
				if err == nil || !strings.Contains(err.Error(), "confirmation required") {
					t.Fatal("missing non-interactive confirmation failure")
				}
			})
		})
	}
}

func TestPortedMutationsValidateClusterIDBeforeDryRun(t *testing.T) {
	for _, c := range mutationPortCases {
		if c.command.Flags().Lookup("cluster-id") == nil {
			continue
		}
		t.Run(c.command.Name(), func(t *testing.T) {
			root := mutationPortRoot(t, c.command)
			args := append([]string{c.command.Name()}, c.args...)
			root.SetArgs(append(args, "--cluster-id", "../fixture", "--dry-run"))
			if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "cluster-id") {
				t.Fatal("invalid path ID accepted")
			}
		})
	}
}

func TestPortedMutationWireAndNoReplay(t *testing.T) {
	for _, c := range mutationPortCases {
		for _, mode := range []string{"machine", "user"} {
			for _, failure := range []int{0, 401, 503} {
				t.Run(fmt.Sprintf("%s/%s/%d", c.command.Name(), mode, failure), func(t *testing.T) {
					var calls atomic.Int32
					server := testutil.ProfileServer(t, mode, "HCM-3", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Method != c.method || r.URL.Path != c.path || r.URL.RawQuery != "" {
							t.Error("incorrect mutation route")
						}
						raw, err := io.ReadAll(r.Body)
						if err != nil {
							t.Error(err)
						}
						if c.body == nil {
							if len(raw) != 0 {
								t.Error("unexpected mutation body")
							}
						} else {
							var got map[string]any
							if err := json.Unmarshal(raw, &got); err != nil {
								t.Error(err)
							}
							if !reflect.DeepEqual(got, c.body) {
								t.Error("mutation body differs from public contract")
							}
						}
						status := c.status
						if failure != 0 {
							status = failure
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(status)
						if failure != 0 {
							fmt.Fprint(w, `{"error":{"message":"fixture failure"}}`)
						} else if c.command == resetWorkspaceServiceAccountCmd {
							fmt.Fprint(w, `{"projectId":"fixture-project","serviceAccountId":"fixture-account","status":"ACTIVE"}`)
						}
					})
					root := mutationPortRoot(t, c.command)
					args := append([]string{c.command.Name()}, c.args...)
					args = append(args, "--endpoint-url", server.URL, "--allow-untrusted-endpoint")
					if c.command.Flags().Lookup("force") != nil {
						args = append(args, "--force")
					}
					root.SetArgs(args)
					testutil.CaptureStdout(t, func() {
						err := root.Execute()
						if (err != nil) != (failure != 0) {
							t.Errorf("mutation result mismatch: %v", err)
						}
					})
					if calls.Load() != 1 {
						t.Fatal("mutation replayed")
					}
				})
			}
		}
	}
}

func TestFleetRegistrationValidationAndOptionalFields(t *testing.T) {
	for _, fleetType := range []string{"NEW", "EXISTING"} {
		body, err := buildFleetBody(fleetType, "", "", "", "", nil)
		if err != nil || !reflect.DeepEqual(body, map[string]any{"fleetType": fleetType}) {
			t.Fatal("optional fleet fields changed")
		}
	}
	body, err := buildFleetBody("EXISTING", "fixture/id:opaque", "fixture-fleet", "disabled", "enabled", map[string]bool{"fleet-id": true, "fleet-name": true, "enable-east-west-traffic": true, "enable-north-south-traffic": true})
	if err != nil || body["id"] != "fixture/id:opaque" || body["enableEastWestTraffic"] != false || body["enableNorthSouthTraffic"] != true {
		t.Fatal("fleet body contract changed")
	}
	for _, c := range []struct {
		kind, name, east, west string
		changed                map[string]bool
	}{
		{"INVALID", "", "", "", nil},
		{"NEW", "BAD NAME", "", "", map[string]bool{"fleet-name": true}},
		{"NEW", "abcd", "", "", map[string]bool{"fleet-name": true}},
		{"NEW", strings.Repeat("a", 21), "", "", map[string]bool{"fleet-name": true}},
		{"NEW", "", "true", "", map[string]bool{"enable-east-west-traffic": true}},
		{"NEW", "", "", "false", map[string]bool{"enable-north-south-traffic": true}},
	} {
		if _, err := buildFleetBody(c.kind, "", c.name, c.east, c.west, c.changed); err == nil {
			t.Fatal("invalid fleet configuration accepted")
		}
	}
}
