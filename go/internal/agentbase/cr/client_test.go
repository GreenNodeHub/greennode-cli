package cr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/client"
)

// fakeTokenProvider satisfies coreclient.TokenProvider so the cr tests do not
// spin up a real IAM token server. The Bearer seam is covered in
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

// TestGetRepository verifies GET /api/v1/repository and the Repository decode.
func TestGetRepository(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repository" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Repository{
			Name: "u-123", RegistryURL: "https://registry.vngcloud.vn/u-123",
			ImageCount: 3, QuotaUsed: 1024, QuotaLimit: 51200,
		})
	})
	out, err := c.GetRepository(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "u-123" || out.ImageCount != 3 || out.QuotaLimit != 51200 {
		t.Errorf("unexpected: %+v", out)
	}
}

// TestListImages verifies the {data,…} envelope decodes and the query uses
// ?page=&size=&name= (camelCase size key, unlike policy's page_size).
func TestListImages(t *testing.T) {
	var gotPage, gotSize, gotName string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repository/images" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotPage = r.URL.Query().Get("page")
		gotSize = r.URL.Query().Get("size")
		gotName = r.URL.Query().Get("name")
		_ = json.NewEncoder(w).Encode(ListImagesResponse{
			Data: []Image{{Name: "myapp", ArtifactCount: 5, PullCount: 12}},
			Page: 2, PageSize: 20, TotalPage: 3, TotalItem: 55,
		})
	})
	out, err := c.ListImages(context.Background(), "mya", 2, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != "2" || gotSize != "20" || gotName != "mya" {
		t.Errorf("query page=%q size=%q name=%q", gotPage, gotSize, gotName)
	}
	if len(out.Data) != 1 || out.Data[0].Name != "myapp" {
		t.Errorf("unexpected data: %+v", out.Data)
	}
	if out.TotalItem != 55 || out.TotalPage != 3 {
		t.Errorf("paging mismatch: %+v", out)
	}
}

func TestListImages_OmittedPagingNotSent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Encode(); q != "" {
			t.Errorf("expected no query when paging/name omitted, got %q", q)
		}
		_ = json.NewEncoder(w).Encode(ListImagesResponse{})
	})
	if _, err := c.ListImages(context.Background(), "", 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteImage verifies DELETE identifies the target via the ?imageName=
// query param (not a path segment) and tolerates the 204 No Content body.
func TestDeleteImage(t *testing.T) {
	var gotMethod, gotImageName string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.URL.Path != "/api/v1/repository/images" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotImageName = r.URL.Query().Get("imageName")
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteImage(context.Background(), "myapp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method=%s, want DELETE", gotMethod)
	}
	if gotImageName != "myapp" {
		t.Errorf("imageName=%q, want myapp", gotImageName)
	}
}

// TestListArtifacts verifies the imageName query is required and the envelope
// decodes (including nested tags on an artifact).
func TestListArtifacts(t *testing.T) {
	var gotImageName, gotName, gotPage, gotSize string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repository/artifacts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotImageName = r.URL.Query().Get("imageName")
		gotName = r.URL.Query().Get("name")
		gotPage = r.URL.Query().Get("page")
		gotSize = r.URL.Query().Get("size")
		_ = json.NewEncoder(w).Encode(ListArtifactsResponse{
			Data: []Artifact{{
				Digest: "sha256:abc", Type: "IMAGE", Size: 2048,
				Tags: []Tag{{Name: "v1"}},
			}},
			Page: 1, PageSize: 10, TotalPage: 1, TotalItem: 1,
		})
	})
	out, err := c.ListArtifacts(context.Background(), "myapp", "sha", 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotImageName != "myapp" || gotName != "sha" || gotPage != "1" || gotSize != "10" {
		t.Errorf("query imageName=%q name=%q page=%q size=%q", gotImageName, gotName, gotPage, gotSize)
	}
	if len(out.Data) != 1 || out.Data[0].Digest != "sha256:abc" {
		t.Errorf("unexpected data: %+v", out.Data)
	}
	if len(out.Data[0].Tags) != 1 || out.Data[0].Tags[0].Name != "v1" {
		t.Errorf("unexpected tags: %+v", out.Data[0].Tags)
	}
}

// TestDeleteArtifact verifies DELETE carries both ?imageName= and ?digest= and
// tolerates 204 No Content.
func TestDeleteArtifact(t *testing.T) {
	var gotImageName, gotDigest string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repository/artifacts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotImageName = r.URL.Query().Get("imageName")
		gotDigest = r.URL.Query().Get("digest")
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.DeleteArtifact(context.Background(), "myapp", "sha256:abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotImageName != "myapp" || gotDigest != "sha256:abc" {
		t.Errorf("imageName=%q digest=%q", gotImageName, gotDigest)
	}
}

// TestGetRegistryCredential verifies GET /api/v1/registry-credential returns the
// robot account (the secret is real — the command layer masks it in tables).
func TestGetRegistryCredential(t *testing.T) {
	var gotAuth string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/registry-credential" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(RegistryCredential{Username: "robot$u-123", Secret: "s3cr3t"})
	})
	out, err := c.GetRegistryCredential(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Username != "robot$u-123" || out.Secret != "s3cr3t" {
		t.Errorf("unexpected: %+v", out)
	}
	if gotAuth != "Bearer test" {
		t.Errorf("Authorization=%q, want Bearer test", gotAuth)
	}
}

// TestResetSecret verifies PATCH /secret sends no body and returns the rotated
// credential (username unchanged, new secret).
func TestResetSecret(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/registry-credential/secret" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if b := mustReadBody(r); len(b) != 0 {
			t.Errorf("reset-secret sends no body, got %q", b)
		}
		_ = json.NewEncoder(w).Encode(RegistryCredential{Username: "robot$u-123", Secret: "n3w"})
	})
	out, err := c.ResetSecret(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Username != "robot$u-123" || out.Secret != "n3w" {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestGetRepository_404ReturnsAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	_, err := c.GetRepository(context.Background())
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

func mustReadBody(r *http.Request) []byte {
	b, _ := io.ReadAll(r.Body)
	return b
}
