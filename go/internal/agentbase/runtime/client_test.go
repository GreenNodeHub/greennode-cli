package runtime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/client"
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
			ListData:  []AgentRuntime{{ID: "1", Name: "rt-one", Status: "ACTIVE"}},
			Page:      1, PageSize: 10, TotalPage: 1, TotalItem: 1,
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
		Name:        "rt",
		ImageURL:    "img:1",
		FlavorID:    "f1",
		Command:     []string{"run"},
		Args:        []string{},
		EnvironmentVariables: map[string]string{"K": "V"},
		ImageAuth:   &ImageAuth{Enabled: true, Username: "u", Password: pw},
		Autoscaling: Autoscaling{MinReplicas: 1, MaxReplicas: 3, CPUUtilization: 70, MemoryUtilization: 70},
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
		Description: "changed",
		ImageURL:    "img:2",
		FlavorID:    "f1",
		Command:     []string{"run"},
		Args:        []string{},
		EnvironmentVariables: map[string]string{"K": "V"},
		Autoscaling: Autoscaling{MinReplicas: 1, MaxReplicas: 2, CPUUtilization: 70, MemoryUtilization: 70},
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
