package vlbclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
)

// stubToken satisfies client.TokenProvider without contacting IAM, so these
// tests exercise the real HTTP client against a local server.
type stubToken struct{}

func (stubToken) GetToken() (string, error)     { return "test-token", nil }
func (stubToken) RefreshToken() (string, error) { return "test-token", nil }

type captured struct {
	method string
	path   string
	query  url.Values
	rawURL string
	body   map[string]interface{}
}

// newTestServer returns a client pointed at a server that records the request
// it receives and replies with the given status and JSON body.
func newTestServer(t *testing.T, status int, response string) (*client.GreennodeClient, *captured) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		got.query = r.URL.Query()
		got.rawURL = r.URL.String()
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &got.body)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want a bearer token", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)

	return client.NewGreennodeClient(srv.URL, stubToken{}, 5*time.Second, 5*time.Second, true, false), got
}

func TestListQueryReachesTheServerDecoded(t *testing.T) {
	c, got := newTestServer(t, 200, `{"listData":[],"page":1,"totalItem":0}`)

	// The shape cmd/vlb/loadbalancer.buildListQuery produces: 1-based page,
	// `size` rather than `pageSize`, and comma-joined array filters.
	params := map[string]string{
		"page": "2", "size": "10", "name": "web",
		"ids": "lb-1,lb-2", "status": "ACTIVE,ERROR",
	}
	if _, err := c.Get(LoadBalancers(proj), params); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.method != http.MethodGet {
		t.Errorf("method = %s, want GET", got.method)
	}
	if got.path != "/v2/proj-1/loadBalancers" {
		t.Errorf("path = %s", got.path)
	}
	for key, want := range params {
		// The commas are percent-encoded on the wire; what matters is that the
		// server decodes them back to a single comma-separated value, which is
		// what Spring binds to a List<String>.
		if v := got.query.Get(key); v != want {
			t.Errorf("query %s = %q, want %q (raw URL: %s)", key, v, want, got.rawURL)
		}
	}
	if got.query.Get("pageSize") != "" {
		t.Error("sent pageSize; VLB uses size (pageSize is the VKS spelling)")
	}
}

func TestPostSendsTheBodyVerbatim(t *testing.T) {
	c, got := newTestServer(t, 200, `{"uuid":"pool-9"}`)

	body := map[string]interface{}{
		"poolName":     "web",
		"poolProtocol": "HTTP",
		"algorithm":    "ROUND_ROBIN",
		"stickiness":   true,
		"healthMonitor": map[string]interface{}{
			"healthCheckProtocol": "HTTP",
			"timeout":             5,
		},
		"members": []map[string]interface{}{
			{"ipAddress": "10.0.0.4", "port": 80, "backup": false},
		},
	}
	res, err := c.Post(Pools(proj, lbID), body)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %s, want POST", got.method)
	}
	if got.path != "/v2/proj-1/loadBalancers/lb-1/pools" {
		t.Errorf("path = %s", got.path)
	}
	// Integers must survive as JSON numbers, not strings — the API rejects
	// "timeout": "5".
	hm := got.body["healthMonitor"].(map[string]interface{})
	if hm["timeout"] != float64(5) {
		t.Errorf("healthMonitor.timeout = %#v, want the JSON number 5", hm["timeout"])
	}
	members := got.body["members"].([]interface{})
	member := members[0].(map[string]interface{})
	if member["port"] != float64(80) || member["backup"] != false {
		t.Errorf("member = %#v", member)
	}
	if got.body["stickiness"] != true {
		t.Errorf("stickiness = %#v, want the JSON boolean true", got.body["stickiness"])
	}

	// A create returns a bare id map, not a full resource.
	if m, ok := res.(map[string]interface{}); !ok || m["uuid"] != "pool-9" {
		t.Errorf("response = %#v", res)
	}
}

func TestPutAndDeleteHitTheRightRoutes(t *testing.T) {
	tests := []struct {
		name       string
		call       func(*client.GreennodeClient) (interface{}, error)
		wantMethod string
		wantPath   string
	}{
		{
			name: "listener update",
			call: func(c *client.GreennodeClient) (interface{}, error) {
				return c.Put(Listener(proj, lbID, listener), map[string]interface{}{"timeoutClient": 60})
			},
			wantMethod: http.MethodPut,
			wantPath:   "/v2/proj-1/loadBalancers/lb-1/listeners/li-1",
		},
		{
			name: "member replacement",
			call: func(c *client.GreennodeClient) (interface{}, error) {
				return c.Put(Members(proj, lbID, poolID), map[string]interface{}{"members": []interface{}{}})
			},
			wantMethod: http.MethodPut,
			wantPath:   "/v2/proj-1/loadBalancers/lb-1/pools/pool-1/members",
		},
		{
			name:       "certificate delete",
			call:       func(c *client.GreennodeClient) (interface{}, error) { return c.Delete(Certificate(proj, caID), nil) },
			wantMethod: http.MethodDelete,
			wantPath:   "/v2/proj-1/cas/ca-1",
		},
		{
			name: "policy delete",
			call: func(c *client.GreennodeClient) (interface{}, error) {
				return c.Delete(Policy(proj, lbID, listener, policyID), nil)
			},
			wantMethod: http.MethodDelete,
			wantPath:   "/v2/proj-1/loadBalancers/lb-1/listeners/li-1/l7policies/pol-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := newTestServer(t, 200, `{}`)
			if _, err := tt.call(c); err != nil {
				t.Fatalf("call: %v", err)
			}
			if got.method != tt.wantMethod {
				t.Errorf("method = %s, want %s", got.method, tt.wantMethod)
			}
			if got.path != tt.wantPath {
				t.Errorf("path = %s, want %s", got.path, tt.wantPath)
			}
		})
	}
}

// TestResponseEnvelopesParse covers the three shapes VLB returns, which the
// command packages' unwrapData helpers and the completers depend on.
func TestResponseEnvelopesParse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		check    func(*testing.T, interface{})
	}{
		{
			name:     "single resource under data",
			response: `{"data":{"uuid":"lb-1","name":"my-lb","displayStatus":"ACTIVE"}}`,
			check: func(t *testing.T, res interface{}) {
				data := res.(map[string]interface{})["data"].(map[string]interface{})
				if data["uuid"] != "lb-1" {
					t.Errorf("data.uuid = %v", data["uuid"])
				}
			},
		},
		{
			name:     "nested list under data",
			response: `{"data":[{"uuid":"li-1"},{"uuid":"li-2"}]}`,
			check: func(t *testing.T, res interface{}) {
				if n := len(res.(map[string]interface{})["data"].([]interface{})); n != 2 {
					t.Errorf("data length = %d, want 2", n)
				}
			},
		},
		{
			name:     "paginated list under listData",
			response: `{"listData":[{"uuid":"lb-1"}],"page":1,"pageSize":10,"totalItem":1}`,
			check: func(t *testing.T, res interface{}) {
				m := res.(map[string]interface{})
				if len(m["listData"].([]interface{})) != 1 || m["totalItem"] != float64(1) {
					t.Errorf("envelope = %#v", m)
				}
			},
		},
		{
			name:     "bare array",
			response: `[{"id":1,"eventType":"SCALE_OUT"}]`,
			check: func(t *testing.T, res interface{}) {
				if len(res.([]interface{})) != 1 {
					t.Errorf("array = %#v", res)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestServer(t, 200, tt.response)
			res, err := c.Get(LoadBalancers(proj), nil)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			tt.check(t, res)
		})
	}
}

// TestExtractCompletionsAgainstRealEnvelopes pins the envelope key each
// completer passes, since picking the wrong one silently yields no suggestions.
func TestExtractCompletionsAgainstRealEnvelopes(t *testing.T) {
	tests := []struct {
		name     string
		response string
		listKeys []string
		want     []string
	}{
		{
			name:     "load balancers use listData",
			response: `{"listData":[{"uuid":"lb-1","name":"prod"},{"uuid":"lb-2","name":"stg"}]}`,
			listKeys: []string{"listData"},
			want:     []string{"lb-1\tprod", "lb-2\tstg"},
		},
		{
			name:     "listeners use data",
			response: `{"data":[{"uuid":"li-1","name":"http-80"}]}`,
			listKeys: []string{"data"},
			want:     []string{"li-1\thttp-80"},
		},
		{
			name:     "items without a name fall back to the bare id",
			response: `{"data":[{"uuid":"pool-1"}]}`,
			listKeys: []string{"data"},
			want:     []string{"pool-1"},
		},
		{
			name:     "the wrong envelope key yields nothing rather than erroring",
			response: `{"listData":[{"uuid":"lb-1","name":"prod"}]}`,
			listKeys: []string{"data"},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded interface{}
			if err := json.Unmarshal([]byte(tt.response), &decoded); err != nil {
				t.Fatal(err)
			}
			got := extractForTest(decoded, tt.listKeys)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("completions = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAPIErrorsCarryTheServerMessage(t *testing.T) {
	c, _ := newTestServer(t, 404, `{"message":"Load balancer not found"}`)

	_, err := c.Get(LoadBalancer(proj, "lb-missing"), nil)
	if err == nil {
		t.Fatal("expected an error for a 404")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *client.APIError", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if !strings.Contains(err.Error(), "Load balancer not found") {
		t.Errorf("error should surface the server message, got: %v", err)
	}
}
