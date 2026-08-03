package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeTokenProvider satisfies coreclient.TokenProvider (GetToken/RefreshToken)
// structurally, so catalog tests do not spin up a real IAM token server. The
// Bearer-header seam itself is covered in internal/agentbase/client.
type fakeTokenProvider struct{ token string }

func (f *fakeTokenProvider) GetToken() (string, error)     { return f.token, nil }
func (f *fakeTokenProvider) RefreshToken() (string, error) { return f.token, nil }

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, &fakeTokenProvider{token: "test"}), srv
}

func wantMethodPath(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("method: got %s, want %s", r.Method, method)
	}
	if r.URL.Path != path {
		t.Errorf("path: got %s, want %s", r.URL.Path, path)
	}
}

func TestListFlavors(t *testing.T) {
	resp := []FlavorEntity{{ID: "f1", Name: "small", CPU: 2, RAM: 4, SupportedResourceTypes: []string{"ai-agent"}, Enabled: true}}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/flavors")
		if got := r.URL.Query().Get("resourceType"); got != "ai-agent" {
			t.Errorf("resourceType: got %q, want ai-agent", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	out, err := c.ListFlavors(context.Background(), "ai-agent")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "f1" || out[0].CPU != 2 {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestListOpenClawVersions(t *testing.T) {
	resp := []OpenClawVersionDto{{ID: "v1", Name: "1.0", DefaultVersion: true}}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/openclaw-versions")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	out, err := c.ListOpenClawVersions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || !out[0].DefaultVersion {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestListOpenClaws(t *testing.T) {
	resp := ListResponseOpenClawDto{ListData: []OpenClawDto{{ID: "oc1", Name: "bot"}}, Page: 1, PageSize: 10, TotalPage: 1, TotalItem: 1}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/openclaws")
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page: got %q, want 2", got)
		}
		if got := r.URL.Query().Get("size"); got != "20" {
			t.Errorf("size: got %q, want 20", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	out, err := c.ListOpenClaws(context.Background(), 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.TotalItem != 1 {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestCreateOpenClaw(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/openclaws")
		var b OpenClawCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if b.Name != "my-bot" || b.VersionID != "v1" || b.FlavorID != "f1" {
			t.Errorf("body: %+v", b)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OpenClawDto{ID: "oc1", Name: b.Name, Status: "ACTIVE"})
	})
	out, err := c.CreateOpenClaw(context.Background(), &OpenClawCreateRequest{Name: "my-bot", VersionID: "v1", FlavorID: "f1"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "oc1" || out.Status != "ACTIVE" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestGetOpenClaw(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/openclaws/oc1")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OpenClawDto{ID: "oc1", Name: "bot"})
	})
	out, err := c.GetOpenClaw(context.Background(), "oc1")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "oc1" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestDeleteOpenClaw(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodDelete, "/v1/openclaws/oc1")
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteOpenClaw(context.Background(), "oc1"); err != nil {
		t.Fatal(err)
	}
}

func TestStartOpenClaw(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/openclaws/oc1/start")
		if r.Body != nil && r.ContentLength > 0 {
			t.Errorf("expected no body, got %d bytes", r.ContentLength)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.StartOpenClaw(context.Background(), "oc1"); err != nil {
		t.Fatal(err)
	}
}

func TestStopOpenClaw(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/openclaws/oc1/stop")
		w.WriteHeader(http.StatusOK)
	})
	if err := c.StopOpenClaw(context.Background(), "oc1"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOpenClawVersion(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPut, "/v1/openclaws/oc1/version")
		if got := r.URL.Query().Get("versionId"); got != "v2" {
			t.Errorf("versionId: got %q, want v2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OpenClawDto{ID: "oc1", VersionID: "v2"})
	})
	out, err := c.UpdateOpenClawVersion(context.Background(), "oc1", "v2")
	if err != nil {
		t.Fatal(err)
	}
	if out.VersionID != "v2" {
		t.Errorf("unexpected decode: %+v", out)
	}
}
