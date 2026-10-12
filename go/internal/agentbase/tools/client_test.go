package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeTokenProvider satisfies coreclient.TokenProvider (GetToken/RefreshToken)
// structurally, so tools tests do not spin up a real IAM token server. The
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

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// wantEndUserHeader asserts the optional X-End-User-ID header: present with
// endUserID when set, absent when empty.
func wantEndUserHeader(t *testing.T, r *http.Request, endUserID string) {
	t.Helper()
	if got := r.Header.Get("X-End-User-ID"); got != endUserID {
		t.Errorf("X-End-User-ID: got %q, want %q", got, endUserID)
	}
}

// --- Browsers ---

func TestListBrowsers(t *testing.T) {
	resp := BrowserPage{ListData: []Browser{{ID: "br1", Name: "web", Status: "ACTIVE"}}, Page: 1, PageSize: 10, TotalItem: 1, TotalPage: 1}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/browsers")
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page: got %q, want 2", got)
		}
		if got := r.URL.Query().Get("size"); got != "20" {
			t.Errorf("size: got %q, want 20", got)
		}
		writeJSON(t, w, resp)
	})
	out, err := c.ListBrowsers(context.Background(), 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].ID != "br1" || out.TotalItem != 1 {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestListBrowsers_DefaultsOmitted(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "" {
			t.Errorf("query: got %q, want empty (defaults omitted)", got)
		}
		writeJSON(t, w, BrowserPage{})
	})
	if _, err := c.ListBrowsers(context.Background(), 0, 0); err != nil {
		t.Fatal(err)
	}
}

func TestCreateBrowser(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/browsers")
		var b CreateBrowserRequest
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if b.Name != "my-browser" || b.Description != "demo" {
			t.Errorf("body: %+v", b)
		}
		writeJSON(t, w, Browser{ID: "br1", Name: b.Name, Status: "CREATING"})
	})
	out, err := c.CreateBrowser(context.Background(), &CreateBrowserRequest{Name: "my-browser", Description: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "br1" || out.Status != "CREATING" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestGetBrowser(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/browsers/br1")
		writeJSON(t, w, Browser{ID: "br1", Name: "web", Status: "ACTIVE"})
	})
	out, err := c.GetBrowser(context.Background(), "br1")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "br1" || out.Status != "ACTIVE" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestDeleteBrowser(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodDelete, "/v1/browsers/br1")
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteBrowser(context.Background(), "br1"); err != nil {
		t.Fatal(err)
	}
}

// --- Browser sessions ---

func TestListBrowserSessions(t *testing.T) {
	resp := BrowserSessionPage{ListData: []BrowserSession{{ID: "s1", Name: "run", Status: "ACTIVE", EndUserID: "alice"}}, Page: 1, PageSize: 10, TotalItem: 1, TotalPage: 1}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/browsers/br1/sessions")
		wantEndUserHeader(t, r, "alice")
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page: got %q, want 2", got)
		}
		writeJSON(t, w, resp)
	})
	out, err := c.ListBrowserSessions(context.Background(), "br1", "alice", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].EndUserID != "alice" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestCreateBrowserSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/browsers/br1/sessions")
		wantEndUserHeader(t, r, "bob")
		var b CreateBrowserSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if b.Name != "run-1" {
			t.Errorf("body: %+v", b)
		}
		writeJSON(t, w, BrowserSession{ID: "s1", Name: b.Name, Status: "ACTIVE", EndUserID: "bob"})
	})
	out, err := c.CreateBrowserSession(context.Background(), "br1", &CreateBrowserSessionRequest{Name: "run-1"}, "bob")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "s1" || out.Status != "ACTIVE" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestGetBrowserSession(t *testing.T) {
	detail := BrowserSessionDetail{
		BrowserSession: BrowserSession{ID: "s1", Name: "run", Status: "ACTIVE"},
		Streams: &BrowserSessionStreams{
			AutomationStream: &BrowserAutomationStream{StreamEndpoint: "https://relay.test/cdp", StreamStatus: "ENABLED"},
			LiveViewStream:   &BrowserLiveViewStream{StreamEndpoint: "https://relay.test/view"},
		},
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/browsers/br1/sessions/s1")
		wantEndUserHeader(t, r, "alice")
		writeJSON(t, w, detail)
	})
	out, err := c.GetBrowserSession(context.Background(), "br1", "s1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if out.Streams == nil || out.Streams.AutomationStream.StreamStatus != "ENABLED" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestDeleteBrowserSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodDelete, "/v1/browsers/br1/sessions/s1")
		wantEndUserHeader(t, r, "")
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteBrowserSession(context.Background(), "br1", "s1", ""); err != nil {
		t.Fatal(err)
	}
}

func TestSignLiveViewStream(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/browsers/br1/sessions/s1/live-view-stream-sign")
		wantEndUserHeader(t, r, "alice")
		writeJSON(t, w, BrowserLiveViewStreamSignature{URL: "https://view.test/s?token=x", ExpiresIn: 300})
	})
	out, err := c.SignLiveViewStream(context.Background(), "br1", "s1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if out.URL != "https://view.test/s?token=x" || out.ExpiresIn != 300 {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestDownloadRecording(t *testing.T) {
	ndjson := "{\"t\":1}\n" + "{\"t\":2}\n"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/browsers/br1/sessions/s1/recording")
		wantEndUserHeader(t, r, "alice")
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(ndjson))
	})
	got, err := c.DownloadRecording(context.Background(), "br1", "s1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != ndjson {
		t.Errorf("raw body: got %q, want %q (verbatim ndjson)", got, ndjson)
	}
}

func TestUpdateBrowserStreams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPut, "/v1/browsers/br1/sessions/s1/streams")
		wantEndUserHeader(t, r, "alice")
		var b UpdateBrowserSessionStreamsRequest
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if b.AutomationStreamUpdate == nil || b.AutomationStreamUpdate.StreamStatus != "DISABLED" {
			t.Errorf("body: %+v", b)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.UpdateBrowserStreams(context.Background(), "br1", "s1", "DISABLED", "alice"); err != nil {
		t.Fatal(err)
	}
}

// --- Code interpreters ---

func TestListCodeInterpreters(t *testing.T) {
	resp := CodeInterpreterPage{ListData: []CodeInterpreter{{ID: "ci1", Name: "py", Status: "ACTIVE"}}, Page: 1, PageSize: 10, TotalItem: 1, TotalPage: 1}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/code-interpreters")
		writeJSON(t, w, resp)
	})
	out, err := c.ListCodeInterpreters(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].ID != "ci1" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestCreateCodeInterpreter(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/code-interpreters")
		var b CreateCodeInterpreterRequest
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if b.Name != "my-ci" {
			t.Errorf("body: %+v", b)
		}
		writeJSON(t, w, CodeInterpreter{ID: "ci1", Name: b.Name, Status: "CREATING"})
	})
	out, err := c.CreateCodeInterpreter(context.Background(), &CreateCodeInterpreterRequest{Name: "my-ci"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "ci1" || out.Status != "CREATING" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestGetAndDeleteCodeInterpreter(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			wantMethodPath(t, r, http.MethodGet, "/v1/code-interpreters/ci1")
			writeJSON(t, w, CodeInterpreter{ID: "ci1", Name: "py", Status: "ACTIVE"})
		case http.MethodDelete:
			wantMethodPath(t, r, http.MethodDelete, "/v1/code-interpreters/ci1")
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	out, err := c.GetCodeInterpreter(context.Background(), "ci1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "ACTIVE" {
		t.Errorf("unexpected decode: %+v", out)
	}
	if err := c.DeleteCodeInterpreter(context.Background(), "ci1"); err != nil {
		t.Fatal(err)
	}
}

func TestListCodeInterpreterSessions(t *testing.T) {
	resp := SessionPage{ListData: []Session{{ID: "s1", Name: "run", Status: "ACTIVE"}}, Page: 1, PageSize: 10, TotalItem: 1, TotalPage: 1}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/v1/code-interpreters/ci1/sessions")
		wantEndUserHeader(t, r, "")
		writeJSON(t, w, resp)
	})
	out, err := c.ListCodeInterpreterSessions(context.Background(), "ci1", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].ID != "s1" {
		t.Errorf("unexpected decode: %+v", out)
	}
}

func TestCreateAndDeleteCodeInterpreterSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantEndUserHeader(t, r, "bob")
		switch r.Method {
		case http.MethodPost:
			wantMethodPath(t, r, http.MethodPost, "/v1/code-interpreters/ci1/sessions")
			var b CreateSessionRequest
			if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if b.Name != "run-1" {
				t.Errorf("body: %+v", b)
			}
			writeJSON(t, w, Session{ID: "s1", Name: b.Name, Status: "ACTIVE", EndUserID: "bob"})
		case http.MethodDelete:
			wantMethodPath(t, r, http.MethodDelete, "/v1/code-interpreters/ci1/sessions/s1")
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	out, err := c.CreateCodeInterpreterSession(context.Background(), "ci1", &CreateSessionRequest{Name: "run-1"}, "bob")
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "s1" || out.Status != "ACTIVE" {
		t.Errorf("unexpected decode: %+v", out)
	}
	if err := c.DeleteCodeInterpreterSession(context.Background(), "ci1", "s1", "bob"); err != nil {
		t.Fatal(err)
	}
}

func TestInvokeCodeInterpreter(t *testing.T) {
	isErr := false
	resp := InvokeResponse{
		Content:           []interface{}{map[string]interface{}{"type": "text", "text": "file: a.txt"}},
		IsError:           &isErr,
		StructuredContent: &map[string]interface{}{"files": []interface{}{"a.txt"}},
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/v1/code-interpreters/ci1/sessions/s1/invoke")
		wantEndUserHeader(t, r, "alice")
		var b InvokeRequest
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if b.Name != "listFiles" || b.Arguments == nil || b.Arguments.DirectoryPath != "/tmp" {
			t.Errorf("body: %+v", b)
		}
		writeJSON(t, w, resp)
	})
	out, err := c.InvokeCodeInterpreter(context.Background(), "ci1", "s1", &InvokeRequest{
		Name:      "listFiles",
		Arguments: &InvokeArguments{DirectoryPath: "/tmp"},
	}, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Content) != 1 || out.IsError == nil || *out.IsError {
		t.Errorf("unexpected decode: %+v", out)
	}
	if out.StructuredContent == nil {
		t.Errorf("structuredContent not decoded")
	}
}
