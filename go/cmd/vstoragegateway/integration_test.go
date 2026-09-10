package vstoragegateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type fixtureToken struct{}

func (fixtureToken) GetToken() (string, error)     { return "fixture-access-token", nil }
func (fixtureToken) RefreshToken() (string, error) { return "fixture-refreshed-token", nil }

func TestAllOperationsThroughHTTPTransport(t *testing.T) {
	fixtures := readCatalog(t)
	for _, op := range allOperations() {
		t.Run(op.Parent, func(t *testing.T) {
			record := fixtures[op.Method+" "+op.Path]
			query := url.Values{}
			for _, p := range record.Parameters {
				if p.In == "query" {
					query.Set(p.Name, "fixture-001")
				}
			}
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != record.Method || r.URL.Path != record.Path || !reflect.DeepEqual(r.URL.Query(), query) {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer fixture-access-token" || r.Header.Get("portal-user-id") != "" {
					t.Error("unexpected auth headers")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, "[]")
			}))
			defer srv.Close()
			previous := newClient
			newClient = func(*cobra.Command) (gatewayAPI, error) {
				return client.NewGreennodeClient(srv.URL, fixtureToken{}, time.Second, time.Second, true, false), nil
			}
			defer func() { newClient = previous }()
			cmd := newOperationCommand(op)
			for _, p := range op.Queries {
				if err := cmd.Flags().Set(p.Flag, "fixture-001"); err != nil {
					t.Fatal(err)
				}
			}
			testutil.CaptureStdout(t, func() {
				if err := cmd.RunE(cmd, nil); err != nil {
					t.Fatal(err)
				}
			})
			if calls.Load() != 1 {
				t.Fatalf("API calls = %d, want 1", calls.Load())
			}
		})
	}
}

func TestGlobalFactoryUsesBothProfileModes(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := testutil.ProfileServer(t, mode, "", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/v1/containers" || r.URL.Query().Get("region_id") != "fixture-region" || r.URL.Query().Get("project_id") != "fixture-project" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, "[]")
			})
			root := testutil.ProfileRoot(newVStorageGatewayCommand())
			root.SetArgs([]string{"vstorage-gateway", "container", "list", "--region-id", "fixture-region", "--project-id", "fixture-project", "--endpoint-url", server.URL, "--allow-untrusted-endpoint"})
			testutil.CaptureStdout(t, func() {
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
			})
			if calls.Load() != 1 {
				t.Fatalf("API calls = %d, want 1", calls.Load())
			}
		})
	}
}
