package vbackup

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
)

type parameterFixture struct {
	In       string `json:"in"`
	Name     string `json:"name"`
	Flag     string `json:"flag"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
	Format   string `json:"format,omitempty"`
	Minimum  int64  `json:"minimum,omitempty"`
}

type operationFixture struct {
	Command     string             `json:"command"`
	OperationID string             `json:"operation_id"`
	Method      string             `json:"method"`
	Path        string             `json:"path"`
	Parameters  []parameterFixture `json:"parameters"`
	Request     *struct {
		Required       bool     `json:"required"`
		MediaType      string   `json:"media_type"`
		Type           string   `json:"type"`
		RequiredFields []string `json:"required_fields"`
		Properties     []string `json:"properties"`
	} `json:"request"`
	Response struct {
		Status    int    `json:"status"`
		MediaType string `json:"media_type"`
		Type      string `json:"type"`
	} `json:"response"`
	Mutation    bool `json:"mutation"`
	Destructive bool `json:"destructive"`
}

func readContractFixtures(t *testing.T) []operationFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion int `json:"schema_version"`
		Source        struct {
			URL, Title, Version, SHA256 string
			RetrievedOn                 string `json:"retrieved_on"`
		} `json:"source"`
		Region         string                          `json:"region"`
		BaseURL        string                          `json:"base_url"`
		Authentication struct{ Header, Scheme string } `json:"authentication"`
		Operations     []operationFixture              `json:"operations"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatal("trailing fixture content")
	}
	if fixture.SchemaVersion != 1 || len(fixture.Operations) != 29 {
		t.Fatal("unsupported or incomplete contract fixture")
	}
	if fixture.Source.URL != "https://docs.api.greennode.ai/service-docs/vbackup-gateway.html" || fixture.Source.Title == "" || fixture.Source.Version == "" {
		t.Fatal("missing official provenance")
	}
	if _, err := time.Parse("2006-01-02", fixture.Source.RetrievedOn); err != nil {
		t.Fatal(err)
	}
	if hash, err := hex.DecodeString(fixture.Source.SHA256); err != nil || len(hash) != 32 {
		t.Fatal("invalid source fingerprint")
	}
	if fixture.Region != "HCM-3" || fixture.BaseURL != "https://hcm-3.api.vngcloud.vn/vbackup-gateway" || fixture.Authentication.Header != "Authorization" || fixture.Authentication.Scheme != "Bearer" {
		t.Fatal("unexpected endpoint or authentication contract")
	}
	commands, signatures, ids := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, f := range fixture.Operations {
		sig := f.Method + " " + f.Path
		if f.Command == "" || f.OperationID == "" || commands[f.Command] || signatures[sig] || ids[f.OperationID] {
			t.Fatalf("duplicate or missing fixture identity: %s", f.Command)
		}
		commands[f.Command], signatures[sig], ids[f.OperationID] = true, true, true
	}
	return fixture.Operations
}

func TestDescriptorsMatchPublicContracts(t *testing.T) {
	fixtures := readContractFixtures(t)
	if len(allOperations()) != len(fixtures) {
		t.Fatal("operation inventory differs from official contract")
	}
	seen := map[string]bool{}
	for _, op := range allOperations() {
		key := op.Parent + " " + op.Use
		if seen[key] {
			t.Fatalf("duplicate command %s", key)
		}
		seen[key] = true
	}
	for _, f := range fixtures {
		t.Run(f.Command, func(t *testing.T) {
			op := fixtureOperation(t, f)
			if op.Method != f.Method || op.Path != f.Path || op.Status != f.Response.Status {
				t.Fatal("method, path, or success status differs from fixture")
			}
			if op.Mutation != f.Mutation || op.Destructive != f.Destructive {
				t.Fatal("incorrect safety classification")
			}
			if (op.Body != nil) != (f.Request != nil) {
				t.Fatal("incorrect request body presence")
			}
			if f.Request != nil && (op.Body.Kind != opengine.ObjectBody || !f.Request.Required || len(op.Body.RequiredFields) != len(f.Request.RequiredFields)) {
				t.Fatal("incorrect request body contract")
			}
			got := map[string]bool{}
			for _, p := range op.Paths {
				got["path:"+p.Placeholder] = true
			}
			for _, p := range op.Queries {
				got["query:"+p.WireName] = p.Required
			}
			want := map[string]bool{}
			for _, p := range f.Parameters {
				want[p.In+":"+p.Name] = p.Required
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("parameters = %v, want %v", got, want)
			}
			for _, p := range f.Parameters {
				if p.Type != "integer" {
					continue
				}
				for _, q := range op.Queries {
					if q.WireName == p.Name && (q.Kind != opengine.QueryInteger || q.Minimum != p.Minimum) {
						t.Fatalf("%s integer constraints differ", p.Name)
					}
				}
			}
			cmd := newOperationCommand(op)
			if (cmd.Flags().Lookup("dry-run") != nil) != f.Mutation || (cmd.Flags().Lookup("force") != nil) != f.Destructive {
				t.Fatal("incorrect safety flags")
			}
			if (f.Response.Type == "empty") != op.HasEmptyStatus(f.Response.Status) {
				t.Fatal("incorrect empty-body status")
			}
			if len(op.EmptySuccess) > 1 {
				t.Fatal("unexpected empty-body status")
			}
		})
	}
}

func fixtureOperation(t *testing.T, f operationFixture) opengine.Descriptor {
	t.Helper()
	parts := strings.Split(f.Command, " ")
	return findOperation(t, parts[0], parts[1])
}

func TestResponseContracts(t *testing.T) {
	for _, f := range readContractFixtures(t) {
		t.Run(f.Command, func(t *testing.T) {
			op := fixtureOperation(t, f)
			var data any = map[string]any{}
			if f.Response.Type == "array" {
				data = []any{}
			}
			valid := client.HTTPResponse{StatusCode: f.Response.Status, Empty: f.Response.Type == "empty", Data: data}
			if err := responseError(op, valid); err != nil {
				t.Fatal(err)
			}
			unexpected := valid
			unexpected.StatusCode = http.StatusAccepted
			if err := responseError(op, unexpected); err == nil {
				t.Fatal("undocumented success status accepted")
			}
			if f.Response.Type != "empty" {
				invalid := valid
				invalid.Empty = true
				if err := responseError(op, invalid); err == nil {
					t.Fatal("empty JSON response accepted")
				}
				invalid.Empty, invalid.Data = false, nil
				if err := responseError(op, invalid); err == nil {
					t.Fatal("null response accepted")
				}
				invalid.Data = "unexpected scalar"
				if err := responseError(op, invalid); err == nil {
					t.Fatal("scalar response accepted")
				}
				invalid.Data = []any{}
				if f.Response.Type == "array" {
					invalid.Data = map[string]any{}
				}
				if err := responseError(op, invalid); err == nil {
					t.Fatal("incorrect JSON container accepted")
				}
			}
		})
	}
}
