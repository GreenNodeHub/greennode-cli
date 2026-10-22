package output

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCredentialOutputAndMetadata(t *testing.T) {
	t.Cleanup(func() { SetShowSecret(false) })
	for _, show := range []bool{false, true} {
		SetShowSecret(show)
		value := map[string]any{"client_id": "fixture-id", "token_type": "Bearer", "client_secret_expires_at": json.Number("9007199254740993"), "token_endpoint_auth_method": "client_secret_basic", "nested": map[string]any{"clientSecret": "fixture-secret", "gatewayToken": "fixture-secret", "Pri_Key": "fixture-secret", "DB-PASSWORD": "fixture-secret", "apiSecret": "fixture-secret"}}
		out := captureStdout(t, func() {
			if err := JSON(value); err != nil {
				t.Fatal(err)
			}
		})
		if strings.Contains(out, "fixture-secret") != show {
			t.Fatal("credential policy mismatch")
		}
		for _, keep := range []string{"fixture-id", "Bearer", "9007199254740993", "client_secret_basic"} {
			if !strings.Contains(out, keep) {
				t.Fatal("public metadata lost")
			}
		}
		if value["nested"].(map[string]any)["clientSecret"] != "fixture-secret" {
			t.Fatal("source data changed")
		}
	}
}

func TestOpaqueCredentials(t *testing.T) {
	t.Cleanup(func() { SetShowSecret(false) })
	for _, show := range []bool{false, true} {
		SetShowSecret(show)
		for _, value := range []any{"fixture-secret", map[string]any{"data": "fixture-secret"}, map[string]any{"result": []any{"fixture-secret"}}} {
			out := captureStdout(t, func() {
				if err := SecretJSON(value); err != nil {
					t.Fatal(err)
				}
			})
			if strings.Contains(out, "fixture-secret") != show {
				t.Fatal("opaque credential policy mismatch")
			}
		}
		out := captureStdout(t, func() { PrintSecret("fixture-secret") })
		if strings.Contains(out, "fixture-secret") != show {
			t.Fatal("scalar credential policy mismatch")
		}
	}
}

func TestCredentialTables(t *testing.T) {
	t.Cleanup(func() { SetShowSecret(false) })
	for _, show := range []bool{false, true} {
		SetShowSecret(show)
		for _, shape := range []bool{false, true} {
			out := captureStdout(t, func() {
				if shape {
					Table([]string{"Name", "Access Token"}, [][]string{{"fixture-name", "fixture-secret"}})
				} else {
					Table([]string{"Field", "Value"}, [][]string{{"API Key", "fixture-secret"}, {"Token Type", "Bearer"}})
				}
			})
			if strings.Contains(out, "fixture-secret") != show {
				t.Fatal("table credential policy mismatch")
			}
		}
	}
}
