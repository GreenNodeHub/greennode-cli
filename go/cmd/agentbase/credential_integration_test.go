package agentbase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type publicContract struct {
	SchemaVersion int `json:"schema_version"`
	Source        struct {
		URL, Title, Version string
		SHA256              string `json:"sha256"`
	} `json:"source"`
	BaseURL       string `json:"base_url"`
	LegacyRuntime bool   `json:"legacy_runtime_paths_supported"`
	Operations    []struct {
		Method    string   `json:"method"`
		Path      string   `json:"path"`
		Required  []string `json:"required_body_fields"`
		Responses map[string]struct {
			Type   string   `json:"type"`
			Fields []string `json:"fields"`
		} `json:"responses"`
	} `json:"operations"`
}

func publicFixture(t *testing.T, name string) publicContract {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name + "-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var f publicContract
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.SchemaVersion != 1 || f.Source.Title == "" || f.Source.Version == "" || !strings.HasPrefix(f.Source.URL, "https://agentbase.api.vngcloud.vn/") || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(f.Source.SHA256) || len(f.Operations) == 0 {
		t.Fatal("invalid public provenance")
	}
	return f
}

type localTransport func(*http.Request) (*http.Response, error)

func (f localTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func checkPublicResponse(t *testing.T, fixture publicContract, method, path, response string) {
	t.Helper()
	for _, op := range fixture.Operations {
		if op.Method != method || op.Path != path {
			continue
		}
		contract, ok := op.Responses["200"]
		if !ok || contract.Type != "object" {
			t.Fatal("missing public response contract")
		}
		var value map[string]any
		if err := json.Unmarshal([]byte(response), &value); err != nil {
			t.Fatal(err)
		}
		for key := range value {
			found := false
			for _, field := range contract.Fields {
				found = found || field == key
			}
			if !found {
				t.Errorf("response field absent from public contract: %s", key)
			}
		}
		return
	}
	t.Fatal("operation absent from public contract")
}

func existingRoot(t *testing.T) *cobra.Command {
	t.Helper()
	previous := AgentbaseCmd.Parent()
	root := testutil.ProfileRoot(AgentbaseCmd)
	t.Cleanup(func() {
		var reset func(*cobra.Command)
		reset = func(cmd *cobra.Command) {
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				if f.Changed {
					_ = f.Value.Set(f.DefValue)
					f.Changed = false
				}
			})
			cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
				if f.Changed {
					_ = f.Value.Set(f.DefValue)
					f.Changed = false
				}
			})
			for _, child := range cmd.Commands() {
				reset(child)
			}
		}
		reset(root)
		root.RemoveCommand(AgentbaseCmd)
		if previous != nil {
			previous.AddCommand(AgentbaseCmd)
		}
		showSecret = false
		output.SetShowSecret(false)
		output.SetFormat(output.FormatTable)
	})
	return root
}

func TestExistingCredentialCommandsUseExplicitDisclosure(t *testing.T) {
	identity := publicFixture(t, "identity")
	runtime := publicFixture(t, "runtime")
	if !runtime.LegacyRuntime {
		t.Fatal("legacy runtime support lost")
	}
	cases := []struct {
		name, method, path, response string
		args                         []string
		idSecret                     bool
	}{
		{"static-key", "GET", "/identity/api/v1/outbound-auth/api-key-providers/fixture-provider/agent-identities/fixture-agent/api-key", `{"apikey":"fixture-secret"}`, []string{"access", "outbound-auth", "static", "get-key", "fixture-provider", "fixture-agent"}, true},
		{"delegated-key", "POST", "/identity/api/v1/outbound-auth/delegated-api-key-providers/fixture-provider/agent-identities/fixture-agent/api-key", `{"apikey":"fixture-secret","authorizationUrl":"https://fixture.example/authorize?code=fixture-secret","sessionId":"fixture-session"}`, []string{"access", "outbound-auth", "delegated", "get-key", "fixture-provider", "fixture-agent", "--agent-user-id", "fixture-user", "--return-url", "https://fixture.example/return"}, false},
		{"m2m-token", "POST", "/identity/api/v1/outbound-auth/oauth2-providers/fixture-provider/agent-identities/fixture-agent/tokens/m2m", `{"accessToken":"fixture-secret","tokenType":"Bearer"}`, []string{"access", "outbound-auth", "oauth2", "m2m-token", "fixture-provider", "fixture-agent", "--scope", "fixture-scope"}, true},
		{"3lo-token", "POST", "/identity/api/v1/outbound-auth/oauth2-providers/fixture-provider/agent-identities/fixture-agent/tokens/3lo", `{"accessToken":"fixture-secret","authorizationUrl":"https://fixture.example/authorize?code=fixture-secret","tokenType":"Bearer"}`, []string{"access", "outbound-auth", "oauth2", "3lo-token", "fixture-provider", "fixture-agent", "--agent-user-id", "fixture-user", "--return-url", "https://fixture.example/return", "--scope", "fixture-scope"}, true},
		{"registry-get", "GET", "/cr/api/v1/registry-credential", `{"username":"fixture-user","secret":"fixture-secret"}`, []string{"cr", "registry-credential", "get"}, false},
		{"registry-rotate", "PATCH", "/cr/api/v1/registry-credential/secret", `{"username":"fixture-user","secret":"fixture-secret"}`, []string{"cr", "registry-credential", "reset-secret"}, false},
		{"openclaw", "GET", "/runtime/v1/openclaws/fixture-id", `{"id":"fixture-id","gatewayToken":"fixture-secret"}`, []string{"marketplace", "openclaw", "get", "fixture-id"}, false},
	}
	for _, c := range cases {
		statuses := []int{200, 403}
		if c.name != "openclaw" {
			statuses = append(statuses, 401)
		}
		for _, mode := range []string{"machine", "user"} {
			for _, format := range []string{"json", "table", "id"} {
				for _, show := range []bool{false, true} {
					for _, status := range statuses {
						t.Run(fmt.Sprintf("%s/%s/%s/%t/%d", c.name, mode, format, show, status), func(t *testing.T) {
							var calls atomic.Int32
							original := http.DefaultTransport
							server := testutil.ProfileServer(t, mode, "HCM-3", func(w http.ResponseWriter, r *http.Request) {
								calls.Add(1)
								path := c.path
								if mode == "user" {
									path = strings.Replace(path, "/runtime/", "/agent-core-runtime/", 1)
								}
								if r.Method != c.method || r.URL.Path != path {
									t.Errorf("wrong credential request: %s %s", r.Method, r.URL.Path)
								}
								if strings.HasPrefix(c.path, "/identity/") {
									found := false
									for _, op := range identity.Operations {
										want := strings.NewReplacer("{providerName}", "fixture-provider", "{agentIdentityName}", "fixture-agent").Replace(op.Path)
										if op.Method != c.method || "/identity"+want != c.path {
											continue
										}
										found = true
										checkPublicResponse(t, identity, op.Method, op.Path, c.response)
										if len(op.Required) > 0 {
											var body map[string]any
											if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
												t.Error(err)
											}
											for _, field := range op.Required {
												if _, ok := body[field]; !ok {
													t.Errorf("missing published field %s", field)
												}
											}
										}
									}
									if !found {
										t.Error("credential operation absent from public fixture")
									}
								}
								if c.name == "openclaw" {
									checkPublicResponse(t, runtime, "GET", "/v1/openclaws/{id}", c.response)
								}
								w.Header().Set("Content-Type", "application/json")
								w.WriteHeader(status)
								fmt.Fprint(w, c.response)
							})
							redirect := http.DefaultTransport
							http.DefaultTransport = localTransport(func(r *http.Request) (*http.Response, error) {
								if strings.HasPrefix(r.URL.String(), server.URL+"/") {
									return original.RoundTrip(r)
								}
								return redirect.RoundTrip(r)
							})
							t.Cleanup(func() { http.DefaultTransport = redirect })
							root := existingRoot(t)
							args := append([]string{"agentbase"}, c.args...)
							args = append(args, "--endpoint-url", server.URL, "--allow-untrusted-endpoint", "--debug", "-o", format)
							if show {
								args = append(args, "--show-secret")
							}
							root.SetArgs(args)
							var out string
							debug := testutil.CaptureStderr(t, func() {
								out = testutil.CaptureStdout(t, func() {
									err := root.Execute()
									if status >= 400 {
										if err == nil || strings.Contains(err.Error(), "fixture-secret") {
											t.Fatal("unsafe credential error")
										}
									} else if err != nil {
										t.Fatal(err)
									}
								})
							})
							wantSecret := status == 200 && show && (format != "id" || c.idSecret)
							if strings.Contains(out, "fixture-secret") != wantSecret {
								t.Fatal("credential disclosure mismatch")
							}
							if strings.Contains(debug, "fixture-secret") || strings.Contains(debug, "fixture-access-token") {
								t.Fatal("debug leaked credentials")
							}
							if calls.Load() != 1 {
								t.Fatal("unexpected API call count")
							}
						})
					}
				}
			}
		}
	}
}

func TestLegacyRuntimeAndMemoryPathsRetainProfileAuth(t *testing.T) {
	runtime := publicFixture(t, "runtime")
	memory := publicFixture(t, "memory")
	if !runtime.LegacyRuntime || memory.BaseURL != "https://agentbase.api.vngcloud.vn/memory" {
		t.Fatal("public endpoint contract changed")
	}
	for _, mode := range []string{"machine", "user"} {
		for _, group := range []string{"runtime", "memory"} {
			t.Run(mode+"/"+group, func(t *testing.T) {
				original := http.DefaultTransport
				server := testutil.ProfileServer(t, mode, "HCM-3", func(w http.ResponseWriter, r *http.Request) {
					base := "/" + group
					if mode == "user" {
						base = "/agent-core-" + group
					}
					path := "/memories"
					if group == "runtime" {
						path = "/agent-runtimes"
					}
					if r.Method != "GET" || r.URL.Path != base+path {
						t.Error("legacy request changed")
					}
					w.Header().Set("Content-Type", "application/json")
					response := `{"listData":[{"id":"fixture-id","name":"fixture-name"}]}`
					fixture, publicPath := memory, "/memories"
					if group == "runtime" {
						fixture, publicPath = runtime, "/v1/agent-runtimes"
					}
					checkPublicResponse(t, fixture, "GET", publicPath, response)
					io.WriteString(w, response)
				})
				redirect := http.DefaultTransport
				http.DefaultTransport = localTransport(func(r *http.Request) (*http.Response, error) {
					if strings.HasPrefix(r.URL.String(), server.URL+"/") {
						return original.RoundTrip(r)
					}
					return redirect.RoundTrip(r)
				})
				t.Cleanup(func() { http.DefaultTransport = redirect })
				root := existingRoot(t)
				root.SetArgs([]string{"agentbase", group, "list", "--endpoint-url", server.URL, "--allow-untrusted-endpoint", "-o", "json"})
				testutil.CaptureStdout(t, func() {
					if err := root.Execute(); err != nil {
						t.Fatal(err)
					}
				})
			})
		}
	}
}
