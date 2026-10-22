package saasai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type catalogRecord struct {
	ServiceID   string `json:"service_id"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	RequestBody *struct {
		Content map[string]struct {
			Schema struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schema"`
		} `json:"content"`
	} `json:"request_body"`
}

func TestCommandsMatchOfficialCatalogTransports(t *testing.T) {
	catalog := readCatalog(t)
	for signature, contentType := range map[string]string{
		"POST " + speechToTextPath: "multipart/form-data",
		"POST " + textToSpeechPath: "application/json",
	} {
		record, ok := catalog[signature]
		if !ok {
			t.Errorf("catalog does not contain %s", signature)
			continue
		}
		if record.RequestBody == nil || len(record.RequestBody.Content[contentType].Schema.Properties) == 0 {
			t.Errorf("%s request content = %#v, want %s", signature, record.RequestBody, contentType)
		}
	}
	if len(catalog) != 2 {
		t.Fatalf("catalog operation count = %d, want 2", len(catalog))
	}
}

func TestPublishedSpeechFields(t *testing.T) {
	catalog := readCatalog(t)
	for _, tc := range []struct {
		path, media string
		fields      map[string]string
	}{
		{speechToTextPath, "multipart/form-data", map[string]string{
			"encoding_type": `{"type":"string","enum":["wav","mp3"]}`,
			"audio_file":    `{"type":"string","format":"binary"}`,
		}},
		{textToSpeechPath, "application/json", map[string]string{
			"input":       `{"type":"string"}`,
			"speed":       `{"type":"number","minimum":0.8,"maximum":1.2}`,
			"speaker_id":  `{"type":"integer","enum":[0,1,2,3]}`,
			"encode_type": `{"type":"integer","enum":[0,1]}`,
		}},
	} {
		properties := catalog["POST "+tc.path].RequestBody.Content[tc.media].Schema.Properties
		if len(properties) != len(tc.fields) {
			t.Fatalf("%s field count = %d, want %d", tc.path, len(properties), len(tc.fields))
		}
		for name, raw := range tc.fields {
			var got, want any
			if err := json.Unmarshal(properties[name], &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(raw), &want); err != nil {
				t.Fatal(err)
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("%s field %s = %s, want %s", tc.path, name, gotJSON, wantJSON)
			}
		}
	}
}

func readCatalog(t *testing.T) map[string]catalogRecord {
	t.Helper()
	id := "saas_ai"
	data, err := os.ReadFile(filepath.Join("testdata", id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion int `json:"schema_version"`
		Source        struct {
			ID     string `json:"id"`
			URL    string `json:"url"`
			SHA256 string `json:"sha256"`
		} `json:"source"`
		Operations []catalogRecord `json:"operations"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != 1 || fixture.Source.ID != id || len(fixture.Source.SHA256) != 64 || fixture.Source.URL == "" {
		t.Fatal("invalid public fixture provenance")
	}
	result := make(map[string]catalogRecord)
	for _, record := range fixture.Operations {
		signature := record.Method + " " + record.Path
		if _, duplicate := result[signature]; duplicate {
			t.Fatalf("duplicate fixture operation: %s", signature)
		}
		result[signature] = record
	}
	return result
}
