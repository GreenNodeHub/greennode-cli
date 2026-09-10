package vserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/login"
	"github.com/greennodehub/greennode-cli/internal/testutil"
)

type coreReadFixture struct {
	Command     string            `json:"command"`
	OperationID string            `json:"operation_id"`
	Flags       map[string]string `json:"flags"`
	PathValues  map[string]string `json:"path_values"`
	Query       string            `json:"query"`
}

func TestCoreReadWireContractsWithBothAuthModes(t *testing.T) {
	data, err := os.ReadFile("testdata/core-reads.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []coreReadFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 39 {
		t.Fatalf("core read fixture count = %d, want 39", len(fixtures))
	}
	var catalog struct {
		Operations []struct {
			OperationID string `json:"operation_id"`
			Method      string `json:"method"`
			Path        string `json:"path"`
		}
	}
	readVServerFixture(t, &catalog)
	byID := make(map[string]struct{ Method, Path string }, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		byID[operation.OperationID] = struct{ Method, Path string }{operation.Method, operation.Path}
	}
	root := testutil.ProfileRoot(VServerCmd)

	for _, mode := range []string{"machine", "user"} {
		t.Run(mode, func(t *testing.T) {
			profileFixture(t, mode)
			var expectedPath string
			var expectedQuery url.Values
			var calls, tokens atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/token" {
					tokens.Add(1)
					fmt.Fprint(w, `{"access_token":"fixture-access","expires_in":3600,"token_type":"Bearer","refresh_token":"fixture-rotated"}`)
					return
				}
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.EscapedPath() != expectedPath || !reflect.DeepEqual(r.URL.Query(), expectedQuery) {
					t.Errorf("wire mismatch: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer fixture-access" || r.Header.Get("portal-user-id") != "12345" {
					t.Error("incorrect authorization or portal-user-id header")
				}
				fmt.Fprint(w, `{}`)
			}))
			defer srv.Close()

			previous := http.DefaultTransport
			prod, _ := login.TokenURLForEnv("prod")
			dev, _ := login.TokenURLForEnv("dev")
			http.DefaultTransport = localTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != prod && r.URL.String() != dev {
					return nil, fmt.Errorf("unexpected token URL")
				}
				clone := r.Clone(r.Context())
				u, _ := url.Parse(srv.URL + "/token")
				clone.URL, clone.Host = u, u.Host
				return previous.RoundTrip(clone)
			})
			t.Cleanup(func() { http.DefaultTransport = previous })

			for _, fixture := range fixtures {
				operation, ok := byID[fixture.OperationID]
				if !ok || operation.Method != http.MethodGet {
					t.Fatalf("missing GET contract for %s", fixture.OperationID)
				}
				expectedPath = operation.Path
				for name, value := range fixture.PathValues {
					expectedPath = strings.ReplaceAll(expectedPath, "{"+name+"}", url.PathEscape(value))
				}
				if strings.ContainsAny(expectedPath, "{}") {
					t.Fatalf("unresolved fixture path %s", expectedPath)
				}
				expectedQuery, err = url.ParseQuery(fixture.Query)
				if err != nil {
					t.Fatal(err)
				}
				args := []string{"--endpoint-url", srv.URL, "--allow-untrusted-endpoint", "--output", "json", "vserver"}
				args = append(args, strings.Fields(fixture.Command)...)
				names := make([]string, 0, len(fixture.Flags))
				for name := range fixture.Flags {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					args = append(args, "--"+name, fixture.Flags[name])
				}
				root.SetArgs(args)
				before := calls.Load()
				testutil.CaptureStdout(t, func() {
					if err := root.Execute(); err != nil {
						t.Fatalf("%s: %v", fixture.Command, err)
					}
				})
				if calls.Load() != before+1 {
					t.Fatalf("%s sent %d requests", fixture.Command, calls.Load()-before)
				}
			}
			if calls.Load() != 39 || tokens.Load() != calls.Load() {
				t.Fatalf("API calls = %d, token calls = %d", calls.Load(), tokens.Load())
			}
		})
	}
}
