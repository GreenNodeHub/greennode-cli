package vstorage

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type fixtureToken struct{}

func (fixtureToken) GetToken() (string, error)     { return "fixture-access-token", nil }
func (fixtureToken) RefreshToken() (string, error) { return "fixture-refreshed-token", nil }

func TestAllOperationsThroughHTTPTransport(t *testing.T) {
	catalogs := readVStorageCatalogs(t)
	records := map[string]catalogRecord{}
	for _, catalog := range catalogs {
		for signature, record := range catalog {
			records[signature] = record
		}
	}
	for _, op := range allOperations() {
		t.Run(op.Parent+"/"+op.Use, func(t *testing.T) {
			record := records[op.Method+" "+op.Path]
			flags := validOperationFlags(op)
			flags["force"] = "true"
			path := record.Path
			for _, parameter := range op.Paths {
				value := "fixture-001"
				switch parameter.Placeholder {
				case "container", "bucket":
					value = "fixture storage"
				case "object", "directory":
					value = "fixture/report & summary.txt"
				case "rule_name":
					value = "fixture rule"
				}
				flags[parameter.Flag] = value
				path = strings.ReplaceAll(path, "{"+parameter.Placeholder+"}", url.PathEscape(value))
			}
			query := url.Values{}
			for _, parameter := range op.Queries {
				flags[parameter.Flag] = "fixture-001"
			}
			for _, parameter := range record.Parameters {
				if parameter.In == "query" {
					query.Set(parameter.Name, "fixture-001")
				}
			}
			var expectedBody any
			if raw, ok := flags["body"]; ok {
				decoder := json.NewDecoder(strings.NewReader(raw))
				decoder.UseNumber()
				if err := decoder.Decode(&expectedBody); err != nil {
					t.Fatal(err)
				}
			}
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != record.Method || r.URL.EscapedPath() != path || !reflect.DeepEqual(r.URL.Query(), query) {
					t.Errorf("request = %s %s?%s, want %s %s?%s", r.Method, r.URL.EscapedPath(), r.URL.RawQuery, record.Method, path, query.Encode())
				}
				if r.Header.Get("Authorization") != "Bearer fixture-access-token" || r.Header.Get("portal-user-id") != "" {
					t.Error("unexpected authentication headers")
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				var got any
				if len(raw) > 0 {
					decoder := json.NewDecoder(strings.NewReader(string(raw)))
					decoder.UseNumber()
					if err := decoder.Decode(&got); err != nil {
						t.Error(err)
					}
				}
				if !reflect.DeepEqual(got, expectedBody) {
					t.Errorf("body = %#v, want %#v", got, expectedBody)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"success":true,"data":[]}`)
			}))
			defer srv.Close()
			previous := newClient
			newClient = func(*cobra.Command) (vstorageAPI, error) {
				return client.NewGreennodeClient(srv.URL, fixtureToken{}, time.Second, time.Second, true, false), nil
			}
			defer func() { newClient = previous }()
			cmd := newOperationCommand(op)
			if !op.Destructive {
				delete(flags, "force")
			}
			setFlags(t, cmd, flags)
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

func TestMutationTransportDoesNotRetry(t *testing.T) {
	for _, status := range []int{401, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(status) }))
			defer srv.Close()
			previous := newClient
			newClient = func(*cobra.Command) (vstorageAPI, error) {
				return client.NewGreennodeClient(srv.URL, fixtureToken{}, time.Second, time.Second, true, false), nil
			}
			defer func() { newClient = previous }()
			op := findOperation(t, "bucket", "create")
			cmd := newOperationCommand(op)
			setFlags(t, cmd, validOperationFlags(op))
			if err := cmd.RunE(cmd, nil); err == nil {
				t.Fatal("expected API error")
			}
			if calls.Load() != 1 {
				t.Fatalf("API calls = %d, want 1", calls.Load())
			}
		})
	}
}

func TestRegionalFactoriesUseBothProfileModes(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		for _, family := range []struct{ region, group, path string }{
			{"HCM-3", "container", "/api/v1/projects/fixture-project"},
			{"HAN", "bucket", "/api/v1/ceph/projects/fixture-project"},
			{"HCM-4", "bucket", "/api/v1/ceph/projects/fixture-project"},
		} {
			t.Run(mode+"/"+family.region, func(t *testing.T) {
				var calls atomic.Int32
				server := testutil.ProfileServer(t, mode, family.region, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != http.MethodGet || r.URL.Path != family.path {
						t.Errorf("unexpected regional request: %s %s", r.Method, r.URL.Path)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"success":true,"data":[]}`)
				})
				previousEndpoint := config.REGIONS[family.region]["vstorage_endpoint"]
				config.REGIONS[family.region]["vstorage_endpoint"] = server.URL
				t.Cleanup(func() { config.REGIONS[family.region]["vstorage_endpoint"] = previousEndpoint })
				root := testutil.ProfileRoot(newVStorageCommand())
				root.SetArgs([]string{"vstorage", family.group, "list", "--project-id", "fixture-project", "--allow-untrusted-endpoint"})
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
}
