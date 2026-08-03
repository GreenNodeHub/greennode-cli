package memory

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
)

// fakeTokenProvider satisfies coreclient.TokenProvider so the memory tests do
// not spin up a real IAM token server. The Bearer seam is covered in
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
// envelope decodes and paging uses ?page=&size= (same envelope as runtime).
func TestList(t *testing.T) {
	var gotPage, gotSize string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotPage = r.URL.Query().Get("page")
		gotSize = r.URL.Query().Get("size")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListMemoriesResponse{
			ListData: []Memory{{ID: "1", Name: "mem-one", Status: "ACTIVE"}},
			Page:     1, PageSize: 10, TotalPage: 1, TotalItem: 1,
		})
	})
	out, err := c.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != "1" || gotSize != "10" {
		t.Errorf("query page=%q size=%q, want 1/10", gotPage, gotSize)
	}
	if len(out.ListData) != 1 || out.ListData[0].Name != "mem-one" {
		t.Errorf("unexpected listData: %+v", out.ListData)
	}
}

func TestList_OmittedPagingNotSent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Encode(); q != "" {
			t.Errorf("expected no query when paging omitted, got %q", q)
		}
		_ = json.NewEncoder(w).Encode(ListMemoriesResponse{})
	})
	if _, err := c.List(context.Background(), 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Memory{ID: "abc", Name: "m", Status: "ACTIVE", EventExpiryDuration: 30})
	})
	out, err := c.Get(context.Background(), "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "m" || out.EventExpiryDuration != 30 {
		t.Errorf("unexpected: %+v", out)
	}
}

// TestCreate verifies the POST body round-trips the nested strategy list.
func TestCreate(t *testing.T) {
	var gotAuth string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		var got CreateMemoryRequest
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got.Name != "m" || got.EventExpiryDuration != 30 {
			t.Errorf("unexpected request: %+v", got)
		}
		if len(got.LongTermMemoryStrategies) != 1 {
			t.Fatalf("expected 1 strategy, got %d", len(got.LongTermMemoryStrategies))
		}
		s := got.LongTermMemoryStrategies[0]
		if s.Name != "sem" || s.Type != "SEMANTIC" || s.NamespaceTemplate != "/ns" {
			t.Errorf("unexpected strategy: %+v", s)
		}
		_ = json.NewEncoder(w).Encode(Memory{ID: "1", Name: "m", Status: "ACTIVE"})
	})
	out, err := c.Create(context.Background(), &CreateMemoryRequest{
		Name:                "m",
		Description:         "d",
		EventExpiryDuration: 30,
		LongTermMemoryStrategies: []LongTermMemoryStrategy{{
			Name: "sem", Type: "SEMANTIC", NamespaceTemplate: "/ns",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "ACTIVE" {
		t.Errorf("unexpected status: %s", out.Status)
	}
	if gotAuth != "Bearer test" {
		t.Errorf("Authorization=%q, want Bearer test", gotAuth)
	}
}

func TestDelete(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.Delete(context.Background(), "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSearch verifies the :search custom-verb route, the required namespace
// query param, the body, and that the ranked []MemoryRecord (with score)
// decodes — including the snake_case created_at/updated_at fields.
func TestSearch(t *testing.T) {
	var gotNS string
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Google-AIP :search custom verb on the memory-records collection.
		if r.URL.Path != "/memories/abc/memory-records:search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotNS = r.URL.Query().Get("namespace")
		gotBody, _ = io.ReadAll(r.Body)
		// Ranked list (Mem0 populates score). snake_case timestamps.
		_ = json.NewEncoder(w).Encode([]MemoryRecord{{
			ID: "r1", Memory: "prefers dark mode", Score: ptrFloat(0.92),
		}})
	})
	out, err := c.Search(context.Background(), "abc", "/strategies/SEMANTIC/actors/alice",
		&SearchMemoryRecordsRequest{Query: "dark mode", Limit: 100, ScoreThreshold: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotNS != "/strategies/SEMANTIC/actors/alice" {
		t.Errorf("namespace query=%q, want the resolved namespace", gotNS)
	}
	var body SearchMemoryRecordsRequest
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("decode search body: %v", err)
	}
	if body.Query != "dark mode" || body.Limit != 100 {
		t.Errorf("unexpected search body: %+v", body)
	}
	if len(out) != 1 || out[0].Memory != "prefers dark mode" {
		t.Errorf("unexpected results: %+v", out)
	}
	if out[0].Score == nil || *out[0].Score != 0.92 {
		t.Errorf("unexpected score: %+v", out[0].Score)
	}
}

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

func ptrFloat(v float64) *float64 { return &v }

// ----------------------------------------------------------------------------
// Slice 3: sub-resources (actors / sessions / events / strategies / records)
// ----------------------------------------------------------------------------

func TestListActors(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/m1/actors" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("size") != "10" {
			t.Errorf("query: %s", r.URL.Query().Encode())
		}
		_ = json.NewEncoder(w).Encode(ListResponseActorDto{
			ListData: []ActorDto{{MemoryID: "m1", ActorID: "a1", Status: "ACTIVE"}},
			Page:     1, PageSize: 10, TotalPage: 1, TotalItem: 1,
		})
	})
	out, err := c.ListActors(context.Background(), "m1", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].ActorID != "a1" {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestListSessions(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/m1/actors/a1/sessions" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ListResponseSessionDto{
			ListData: []SessionDto{{SessionID: "s1", Status: "ACTIVE"}},
		})
	})
	out, err := c.ListSessions(context.Background(), "m1", "a1", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ListData) != 1 || out.ListData[0].SessionID != "s1" {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestListSessionEvents(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/m1/actors/a1/sessions/s1/events" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("fromTimestamp"); got != "t0" {
			t.Errorf("from: %q", got)
		}
		if got := r.URL.Query().Get("toTimestamp"); got != "t1" {
			t.Errorf("to: %q", got)
		}
		// size clamped to 100.
		if got := r.URL.Query().Get("size"); got != "100" {
			t.Errorf("size: %q, want 100 (clamped)", got)
		}
		// Undefined schema → raw JSON array.
		_ = json.NewEncoder(w).Encode([]map[string]string{
			{"type": "USER", "message": "hi"},
			{"type": "AGENT", "message": "hello"},
		})
	})
	out, err := c.ListSessionEvents(context.Background(), "m1", "a1", "s1", "t0", "t1", 1, 9999)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("events: %d, want 2", len(out))
	}
}

func TestCreateSessionEvent(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/memories/m1/actors/a1/sessions/s1/events" {
			t.Errorf("method/path: %s %s", r.Method, r.URL.Path)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	if err := c.CreateSessionEvent(context.Background(), "m1", "a1", "s1", &EventCreateRequest{
		Payload: EventPayload{Type: "USER", Message: "hi"},
	}); err != nil {
		t.Fatal(err)
	}
	var got EventCreateRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Payload.Type != "USER" || got.Payload.Message != "hi" {
		t.Errorf("body: %+v", got)
	}
}

func TestDeleteSessionEvent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/memories/m1/actors/a1/sessions/s1/events/e1" {
			t.Errorf("method/path: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteSessionEvent(context.Background(), "m1", "a1", "s1", "e1"); err != nil {
		t.Fatal(err)
	}
}

func TestListStrategies(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/m1/long-term-memory-strategies" {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]LongTermMemoryStrategyEntity{{
			ID: "st1", Name: "sem", Type: "SEMANTIC", NamespaceTemplate: "/ns",
		}})
	})
	out, err := c.ListStrategies(context.Background(), "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Type != "SEMANTIC" {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestListRecords(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/m1/memory-records" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("namespace"); got != "/ns" {
			t.Errorf("namespace: %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Errorf("limit: %q", got)
		}
		// snake_case timestamps.
		_ = json.NewEncoder(w).Encode([]MemoryRecord{{ID: "r1", Memory: "fact"}})
	})
	out, err := c.ListRecords(context.Background(), "m1", "/ns", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Memory != "fact" {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestDeleteRecord(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/memories/m1/memory-records/r1" {
			t.Errorf("method/path: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteRecord(context.Background(), "m1", "r1"); err != nil {
		t.Fatal(err)
	}
}

func TestInsertRecords(t *testing.T) {
	var gotBody []byte
	var gotNS string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/memories/m1/memory-records:insert-directly" {
			t.Errorf("method/path: %s %s", r.Method, r.URL.Path)
		}
		gotNS = r.URL.Query().Get("namespace")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	if err := c.InsertRecords(context.Background(), "m1", "/ns", &MemoryRecordInsertDirectlyRequest{
		MemoryRecords: []string{"fact-1", "fact-2"},
	}); err != nil {
		t.Fatal(err)
	}
	if gotNS != "/ns" {
		t.Errorf("namespace: %q", gotNS)
	}
	var got MemoryRecordInsertDirectlyRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.MemoryRecords) != 2 {
		t.Errorf("records: %d", len(got.MemoryRecords))
	}
}

func TestGenerateRecordsFromSession(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/memories/m1/memory-records:generate-from-session" {
			t.Errorf("method/path: %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("actorId") != "a1" || q.Get("sessionId") != "s1" || q.Get("longTermMemoryStrategyId") != "st1" {
			t.Errorf("query: %s", q.Encode())
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	if err := c.GenerateRecordsFromSession(context.Background(), "m1", "a1", "s1", "st1"); err != nil {
		t.Fatal(err)
	}
	if len(gotBody) != 0 {
		t.Errorf("expected no body, got %d bytes", len(gotBody))
	}
}

func TestGenerateRecordsFromContent(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/memories/m1/memory-records:generate-from-content" {
			t.Errorf("method/path: %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("longTermMemoryStrategyId") != "st1" || q.Get("actorId") != "a1" || q.Get("sessionId") != "s1" {
			t.Errorf("query: %s", q.Encode())
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	if err := c.GenerateRecordsFromContent(context.Background(), "m1", "st1", "a1", "s1", &MemoryRecordGenerateFromContentRequest{
		ChatMessages: []ChatMessage{{Role: "user", Content: "hi"}},
	}); err != nil {
		t.Fatal(err)
	}
	var got MemoryRecordGenerateFromContentRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.ChatMessages) != 1 || got.ChatMessages[0].Role != "user" {
		t.Errorf("body: %+v", got)
	}
}
