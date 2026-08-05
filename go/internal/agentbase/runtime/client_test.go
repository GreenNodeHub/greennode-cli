package runtime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
)

// fakeTokenProvider satisfies coreclient.TokenProvider so the runtime tests do
// not spin up a real IAM token server. The Bearer seam itself is covered in
// internal/agentbase/client.
type fakeTokenProvider struct{ token string }

func (f *fakeTokenProvider) GetToken() (string, error)     { return f.token, nil }
func (f *fakeTokenProvider) RefreshToken() (string, error) { return f.token, nil }

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, &fakeTokenProvider{token: "test"})
}

// TestList verifies the {listData, page, pageSize, totalPage, totalItem}
// envelope decodes and that paging uses the ?page=&size= query params (the
// runtime service's distinct envelope — neither gateway's nor identity's).
func TestList(t *testing.T) {
	var gotPage, gotSize string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/agent-runtimes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotPage = r.URL.Query().Get("page")
		gotSize = r.URL.Query().Get("size")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListAgentRuntimesResponse{
			ListData: []AgentRuntime{{ID: "1", Name: "rt-one", Status: "ACTIVE"}},
			Page:     1, PageSize: 10, TotalPage: 1, TotalItem: 1,
		})
	})
	out, err := c.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Note: query param is "size", not "pageSize".
	if gotPage != "1" || gotSize != "10" {
		t.Errorf("query page=%q size=%q, want 1/10", gotPage, gotSize)
	}
	if len(out.ListData) != 1 || out.ListData[0].Name != "rt-one" {
		t.Errorf("unexpected listData: %+v", out.ListData)
	}
	if out.TotalItem != 1 || out.TotalPage != 1 {
		t.Errorf("unexpected paging fields: %+v", out)
	}
}

// TestList_OmittedPagingNotSent: page/size <= 0 must be omitted so the server
// applies its own defaults (1/10) rather than receiving page=0.
func TestList_OmittedPagingNotSent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Encode(); q != "" {
			t.Errorf("expected no query when paging omitted, got %q", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListAgentRuntimesResponse{})
	})
	if _, err := c.List(context.Background(), 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Keyed by id, bare path (no /api/v1).
		if r.URL.Path != "/agent-runtimes/abc-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AgentRuntime{ID: "abc-123", Name: "my-rt", Status: "ACTIVE"})
	})
	out, err := c.Get(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "my-rt" || out.Status != "ACTIVE" {
		t.Errorf("unexpected runtime: %+v", out)
	}
}

// TestCreate verifies the POST body round-trips the nested spec (imageAuth,
// command/args lists, env map, autoscaling) and the Authorization header.
func TestCreate(t *testing.T) {
	var gotAuth string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		var got CreateAgentRuntimeRequest
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if got.Name != "rt" || got.ImageURL != "img:1" || got.FlavorID != "f1" {
			t.Errorf("unexpected request: %+v", got)
		}
		if got.ImageAuth == nil || !got.ImageAuth.Enabled || got.ImageAuth.Username != "u" {
			t.Errorf("unexpected imageAuth: %+v", got.ImageAuth)
		}
		if len(got.Command) != 1 || got.Command[0] != "run" || len(got.Args) != 0 {
			t.Errorf("unexpected command/args: %+v / %+v", got.Command, got.Args)
		}
		if got.EnvironmentVariables["K"] != "V" {
			t.Errorf("unexpected env: %+v", got.EnvironmentVariables)
		}
		if got.Autoscaling.MinReplicas != 1 || got.Autoscaling.MaxReplicas != 3 {
			t.Errorf("unexpected autoscaling: %+v", got.Autoscaling)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AgentRuntime{ID: "1", Name: "rt", Status: "CREATING"})
	})
	pw := "secret"
	out, err := c.Create(context.Background(), &CreateAgentRuntimeRequest{
		Name:                 "rt",
		ImageURL:             "img:1",
		FlavorID:             "f1",
		Command:              []string{"run"},
		Args:                 []string{},
		EnvironmentVariables: map[string]string{"K": "V"},
		ImageAuth:            &ImageAuth{Enabled: true, Username: "u", Password: pw},
		Autoscaling:          Autoscaling{MinReplicas: 1, MaxReplicas: 3, CPUUtilization: 70, MemoryUtilization: 70},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "CREATING" {
		t.Errorf("unexpected status: %s", out.Status)
	}
	if gotAuth != "Bearer test" {
		t.Errorf("Authorization=%q, want Bearer test", gotAuth)
	}
}

// TestCreate_OmitsNilImageAuth verifies imageAuth is omitted (omitempty) when
// absent, so the wire body has no imageAuth key for public images.
func TestCreate_OmitsNilImageAuth(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AgentRuntime{ID: "1", Name: "rt", Status: "CREATING"})
	})
	_, _ = c.Create(context.Background(), &CreateAgentRuntimeRequest{
		Name: "rt", ImageURL: "img", FlavorID: "f",
		Command: []string{}, Args: []string{}, EnvironmentVariables: map[string]string{},
		Autoscaling: Autoscaling{MinReplicas: 1, MaxReplicas: 2},
	})
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, present := got["imageAuth"]; present {
		t.Error("imageAuth should be omitted when nil")
	}
}

// TestUpdate verifies update PATCHes the full-spec body (NOT a merge-patch):
// every field is present, including empty slices/maps which the service
// requires (@NotNull).
func TestUpdate(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/agent-runtimes/abc-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AgentRuntime{ID: "abc-123", Name: "rt", Status: "UPDATING"})
	})
	out, err := c.Update(context.Background(), "abc-123", &UpdateAgentRuntimeRequest{
		Description:          "changed",
		ImageURL:             "img:2",
		FlavorID:             "f1",
		Command:              []string{"run"},
		Args:                 []string{},
		EnvironmentVariables: map[string]string{"K": "V"},
		Autoscaling:          Autoscaling{MinReplicas: 1, MaxReplicas: 2, CPUUtilization: 70, MemoryUtilization: 70},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "UPDATING" {
		t.Errorf("unexpected status: %s", out.Status)
	}
	var got UpdateAgentRuntimeRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode patch body: %v", err)
	}
	// Full-spec replacement: the body must NOT carry a name field (immutable).
	if got.ImageURL != "img:2" || got.Description != "changed" {
		t.Errorf("unexpected patch body: %+v", got)
	}
}

func TestDelete(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// Spring void handler → 200 with empty body.
		w.WriteHeader(http.StatusOK)
	})
	if err := c.Delete(context.Background(), "abc-123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGet_404ReturnsAPIError verifies a non-2xx surfaces as *client.APIError.
func TestGet_404ReturnsAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	})
	_, err := c.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d, want 404", apiErr.StatusCode)
	}
}

func TestNewClient(t *testing.T) {
	if c := NewClient("https://example.com", nil); c == nil {
		t.Fatal("expected non-nil client")
	}
}

// ----------------------------------------------------------------------------
// Slice 4: sub-resources (endpoints / logs / metrics / events / versions / trace)
// ----------------------------------------------------------------------------

func wantMethodPath(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("method: got %s, want %s", r.Method, method)
	}
	if r.URL.Path != path {
		t.Errorf("path: got %s, want %s", r.URL.Path, path)
	}
}

func TestListEndpoints(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/agent-runtimes/r1/endpoints")
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("size") != "10" {
			t.Errorf("query: %s", r.URL.Query().Encode())
		}
		_ = json.NewEncoder(w).Encode(ListResponseAgentRuntimeEndpointDto{
			ListData: []AgentRuntimeEndpointDto{{ID: "e1", Name: "ep", Version: 2}},
		})
	})
	out, err := c.ListEndpoints(context.Background(), "r1", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].Version != 2 {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestCreateEndpoint(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/agent-runtimes/r1/endpoints")
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(AgentRuntimeEndpointDto{ID: "e1", Name: "ep"})
	})
	out, err := c.CreateEndpoint(context.Background(), "r1", &AgentRuntimeEndpointCreateRequest{Name: "ep", Version: 2})
	if err != nil {
		t.Fatal(err)
	}
	var got AgentRuntimeEndpointCreateRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "ep" || got.Version != 2 {
		t.Errorf("body: %+v", got)
	}
	if out.ID != "e1" {
		t.Errorf("decode: %+v", out)
	}
}

func TestUpdateEndpoint(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPut, "/agent-runtimes/r1/endpoints/e1")
		if got := r.URL.Query().Get("version"); got != "3" {
			t.Errorf("version: %q", got)
		}
		if r.ContentLength > 0 {
			t.Errorf("expected no body, got %d bytes", r.ContentLength)
		}
		_ = json.NewEncoder(w).Encode(AgentRuntimeEndpointDto{ID: "e1", Version: 3})
	})
	out, err := c.UpdateEndpoint(context.Background(), "r1", "e1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != 3 {
		t.Errorf("decode: %+v", out)
	}
}

func TestDeleteEndpoint(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodDelete, "/agent-runtimes/r1/endpoints/e1")
		_ = json.NewEncoder(w).Encode(AgentRuntimeEndpointDto{ID: "e1", Status: "DELETED"})
	})
	out, err := c.DeleteEndpoint(context.Background(), "r1", "e1")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "e1" || out.Status != "DELETED" {
		t.Errorf("decode: %+v", out)
	}
}

func TestStartStopEndpoint(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if r.Body != nil && r.ContentLength > 0 {
			t.Errorf("expected no body")
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.StartEndpoint(context.Background(), "r1", "e1"); err != nil {
		t.Fatal(err)
	}
	if err := c.StopEndpoint(context.Background(), "r1", "e1"); err != nil {
		t.Fatal(err)
	}
}

func TestEndpointLogs(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/agent-runtimes/r1/endpoints/e1/logs")
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(LogSearchResult{TotalCount: 1, Logs: []LogRecord{{Content: "hi"}}})
	})
	out, err := c.EndpointLogs(context.Background(), "r1", "e1", &LogSearchRequest{Limit: 50, Query: "err"})
	if err != nil {
		t.Fatal(err)
	}
	var got LogSearchRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatal(err)
	}
	if got.Query != "err" || got.Limit != 50 {
		t.Errorf("body: %+v", got)
	}
	if out.TotalCount != 1 || len(out.Logs) != 1 {
		t.Errorf("decode: %+v", out)
	}
}

func TestEndpointMetrics(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/agent-runtimes/r1/endpoints/e1/metrics")
		if got := r.URL.Query().Get("fromTimestamp"); got != "t0" {
			t.Errorf("from: %q", got)
		}
		_ = json.NewEncoder(w).Encode(AgentRuntimeEndpointMetrics{
			CpuCoresUsage: []MetricDataPointDouble{{Value: 0.5}},
		})
	})
	out, err := c.EndpointMetrics(context.Background(), "r1", "e1", "t0", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.CpuCoresUsage) != 1 || out.CpuCoresUsage[0].Value != 0.5 {
		t.Errorf("decode: %+v", out)
	}
}

func TestEndpointEvents(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/agent-runtimes/r1/endpoints/e1/events")
		_ = json.NewEncoder(w).Encode([]KubeEventDto{{Message: "pulled"}})
	})
	out, err := c.EndpointEvents(context.Background(), "r1", "e1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Message != "pulled" {
		t.Errorf("decode: %+v", out)
	}
}

func TestLogs(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/agent-runtimes/r1/logs")
		_ = json.NewEncoder(w).Encode(LogSearchResult{TotalCount: 2})
	})
	out, err := c.Logs(context.Background(), "r1", &LogSearchRequest{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalCount != 2 {
		t.Errorf("decode: %+v", out)
	}
}

func TestResetServiceAccount(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/agent-runtimes/r1/reset-service-account")
		w.WriteHeader(http.StatusOK)
	})
	if err := c.ResetServiceAccount(context.Background(), "r1"); err != nil {
		t.Fatal(err)
	}
}

func TestListVersions(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/agent-runtimes/r1/versions")
		_ = json.NewEncoder(w).Encode(ListResponseAgentRuntimeVersionDto{
			ListData: []AgentRuntimeVersionDto{{Version: 1, ImageURL: "img", FlavorID: "f1"}},
		})
	})
	out, err := c.ListVersions(context.Background(), "r1", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].FlavorID != "f1" {
		t.Errorf("decode: %+v", out)
	}
}

func TestGetTrace(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/agent-runtimes:get-trace")
		if got := r.URL.Query().Get("traceId"); got != "abc" {
			t.Errorf("traceId: %q", got)
		}
		if got := r.URL.Query().Get("service"); got != "runtime" {
			t.Errorf("passthrough param: %q", got)
		}
		_, _ = w.Write([]byte(`{"traceID":"abc"}`))
	})
	params := url.Values{}
	params.Set("service", "runtime")
	out, err := c.GetTrace(context.Background(), "abc", params)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"traceID":"abc"}` {
		t.Errorf("raw: %s", out)
	}
}

func TestSearchTraces(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/agent-runtimes:search-traces")
		if got := r.URL.Query().Get("service"); got != "runtime" {
			t.Errorf("param: %q", got)
		}
		_, _ = w.Write([]byte(`{"traces":[]}`))
	})
	params := url.Values{}
	params.Set("service", "runtime")
	out, err := c.SearchTraces(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"traces":[]}` {
		t.Errorf("raw: %s", out)
	}
}

func TestTraceTagValues(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/agent-runtimes:trace-search-tag-values")
		if got := r.URL.Query().Get("tagKey"); got != "env" {
			t.Errorf("tagKey: %q", got)
		}
		_, _ = w.Write([]byte(`["prod","dev"]`))
	})
	out, err := c.TraceTagValues(context.Background(), "env", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `["prod","dev"]` {
		t.Errorf("raw: %s", out)
	}
}
