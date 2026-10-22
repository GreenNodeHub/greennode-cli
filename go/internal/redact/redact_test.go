package redact

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestIsSensitiveKeyLegacyCliFragments(t *testing.T) {
	// Every fragment previously recognized by internal/cli.RedactJSON.
	for _, key := range []string{
		"secret", "apiSecret",
		"password", "userPassword",
		"token", "accessToken",
		"privateKey", "private_key",
		"accessKey", "access-key",
		"apiKey", "api_key", "api-key",
		"authorization",
		"credential", "credentials",
		"clientId", "client_id",
	} {
		if !IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = false, want true (legacy cli fragment)", key)
		}
	}
}

func TestIsSensitiveKeyLegacyClientKeys(t *testing.T) {
	// Every key/fragment previously recognized by internal/client.RedactJSON.
	for _, key := range []string{
		"kubeconfig", "kubeConfig",
		"clientCertificateData", "client-certificate-data",
		"userData", "userdata",
		"consoleUrl", "console_url",
		"auth",
		"authorization",
		"authentication",
		"secret", "passphrase", "token", "credential",
		"key", // bare exact match, narrower than the legacy substring
	} {
		if !IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = false, want true (legacy client key)", key)
		}
	}
}

func TestIsSensitiveKeyLegacyVLBFragments(t *testing.T) {
	for _, key := range []string{"secret", "password", "passphrase", "token", "privateKey", "accessKey"} {
		if !IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = false, want true (legacy vlb fragment)", key)
		}
	}
}

func TestIsSensitiveKeyDoesNotFalsePositiveOnKeySubstring(t *testing.T) {
	// "key" in the middle or start of a normalized name (not as a suffix)
	// must not trigger redaction.
	for _, key := range []string{"keyName", "keyname", "keyword", "Keyword", "keypairName", "hockeyTeam"} {
		if IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = true, want false (\"key\" as a non-suffix substring must not match)", key)
		}
	}
}

func TestIsSensitiveKeyKeySuffixCoversCompoundKeyMaterial(t *testing.T) {

	for _, key := range []string{
		"sshKey", "ssh_key", "ssh-key",
		"priKey", "pri_key", "pri-key",
		"authKey",
		"masterKey",
		"hostKey",
		"encryptionKey",
	} {
		if !IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = false, want true (compound key-material field ending in \"key\")", key)
		}
	}
}

func TestIsSensitiveKeyKeyDataFragment(t *testing.T) {
	for _, key := range []string{"clientKeyData", "client-key-data", "keyData", "key_data"} {
		if !IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = false, want true (keydata fragment)", key)
		}
	}
}

func TestIsSensitiveKeyAcceptsOverRedactionOnWordsEndingInKey(t *testing.T) {

	for _, key := range []string{"monkey", "Monkey", "turkey", "hockey", "donkey", "jockey"} {
		if !IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = false, want true (deliberate over-redaction for words ending in \"key\")", key)
		}
	}
}

func TestIsSensitiveKeySafeFields(t *testing.T) {
	for _, key := range []string{"id", "name", "status", "numNodes", "createdAt", "authorName", "author"} {
		if IsSensitiveKey(key) {
			t.Errorf("IsSensitiveKey(%q) = true, want false", key)
		}
	}
}

func TestJSONRedactsNestedStructuresAndArraysOfMaps(t *testing.T) {
	input := map[string]any{
		"apiKey":        "api-key-value",
		"api_key":       "api-key-underscore-value",
		"authorization": "authorization-value",
		"credential":    "credential-value",
		"clientId":      "client-id-value",
		"clientSecret":  "client-secret-value",
		"key":           "bare-key-value",
		"nested": map[string]any{
			"accessToken": "access-token-value",
			"safe":        "visible-value",
		},
		"items": []any{
			map[string]any{"private_key": "private-key-value", "keyName": "not-secret"},
			map[string]any{"password": "password-value"},
		},
		"safe": "visible-top-level-value",
	}

	got, ok := JSON(input).(map[string]any)
	if !ok {
		t.Fatalf("JSON() type = %T, want map[string]any", JSON(input))
	}
	for _, key := range []string{"apiKey", "api_key", "authorization", "credential", "clientId", "clientSecret", "key"} {
		if got[key] != Value {
			t.Errorf("got[%q] = %#v, want %q", key, got[key], Value)
		}
	}
	if got["safe"] != "visible-top-level-value" {
		t.Errorf("safe = %#v, want unchanged", got["safe"])
	}

	nested, ok := got["nested"].(map[string]any)
	if !ok || nested["accessToken"] != Value || nested["safe"] != "visible-value" {
		t.Errorf("nested = %#v, want recursive redaction only on accessToken", got["nested"])
	}

	items, ok := got["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items = %#v, want two-element array", got["items"])
	}
	first := items[0].(map[string]any)
	if first["private_key"] != Value {
		t.Errorf("private_key = %#v, want %q", first["private_key"], Value)
	}
	if first["keyName"] != "not-secret" {
		t.Errorf("keyName = %#v, want unchanged (false positive)", first["keyName"])
	}
	if items[1].(map[string]any)["password"] != Value {
		t.Errorf("password = %#v, want %q", items[1].(map[string]any)["password"], Value)
	}

	// Input must not be mutated.
	if !reflect.DeepEqual(input["nested"], map[string]any{"accessToken": "access-token-value", "safe": "visible-value"}) {
		t.Fatalf("input mutated: %#v", input["nested"])
	}
}

func TestJSONScalarsAndNilPassThrough(t *testing.T) {
	for _, v := range []any{nil, "plain string", 42, true, 3.14} {
		if got := JSON(v); !reflect.DeepEqual(got, v) {
			t.Errorf("JSON(%#v) = %#v, want unchanged", v, got)
		}
	}
}

func TestJSONEmptyMapAndSlice(t *testing.T) {
	if got := JSON(map[string]any{}); !reflect.DeepEqual(got, map[string]any{}) {
		t.Errorf("JSON(empty map) = %#v, want empty map", got)
	}
	if got := JSON([]any{}); !reflect.DeepEqual(got, []any{}) {
		t.Errorf("JSON(empty slice) = %#v, want empty slice", got)
	}
}

func TestBodyRedactsJSONObject(t *testing.T) {
	in := `{"status":"ACTIVE","kubeConfig":"apiVersion: v1\nclient-key-data: SUPERSECRETKEY","renewalWarning":false}`
	out := Body(in)
	if out == in {
		t.Fatalf("Body() did not change JSON input")
	}
	if strings.Contains(out, "SUPERSECRETKEY") {
		t.Errorf("kubeConfig value not redacted: %s", out)
	}
	if !strings.Contains(out, Value) {
		t.Errorf("expected %s marker: %s", Value, out)
	}
	if !strings.Contains(out, "ACTIVE") {
		t.Errorf("non-sensitive field lost: %s", out)
	}
}

func TestBodyNonJSONPassthrough(t *testing.T) {
	for _, in := range []string{"not json", "", "   ", "<xml>secret</xml>"} {
		if got := Body(in); got != in {
			t.Errorf("Body(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestBodyKeepsNonSensitiveJSON(t *testing.T) {
	in := `{"id":"cls-1","numNodes":3,"status":"CREATING"}`
	out := Body(in)
	for _, keep := range []string{"cls-1", "3", "CREATING"} {
		if !strings.Contains(out, keep) {
			t.Errorf("Body() lost non-sensitive value %q: %s", keep, out)
		}
	}
	if strings.Contains(out, Value) {
		t.Errorf("Body() redacted something it shouldn't have: %s", out)
	}
}

func TestURLRedactsSensitiveQueryParameters(t *testing.T) {
	in := "https://api.example.com/v1/things?token=abc123&name=visible&apiKey=xyz&plain=1"
	out := URL(in)
	if strings.Contains(out, "abc123") || strings.Contains(out, "xyz") {
		t.Fatalf("URL() leaked a sensitive value: %s", out)
	}
	if !strings.Contains(out, "name=visible") || !strings.Contains(out, "plain=1") {
		t.Fatalf("URL() dropped or mangled a non-sensitive parameter: %s", out)
	}
	// Order of parameters must be preserved: token, name, apiKey, plain.
	tokenIdx := strings.Index(out, "token=")
	nameIdx := strings.Index(out, "name=")
	apiKeyIdx := strings.Index(out, "apiKey=")
	plainIdx := strings.Index(out, "plain=")
	if !(tokenIdx < nameIdx && nameIdx < apiKeyIdx && apiKeyIdx < plainIdx) {
		t.Fatalf("URL() reordered query parameters: %s", out)
	}
}

func TestURLNoQueryStringPassthrough(t *testing.T) {
	in := "https://api.example.com/v1/things"
	if got := URL(in); got != in {
		t.Errorf("URL(%q) = %q, want unchanged", in, got)
	}
}

func TestURLParseFailurePassthrough(t *testing.T) {
	in := "://bad-url?token=abc123"
	if got := URL(in); got != in {
		t.Errorf("URL(%q) = %q, want unchanged input on parse failure", in, got)
	}
}

func TestURLValuelessSensitiveParamUntouched(t *testing.T) {
	in := "https://api.example.com/v1/things?debug&token"
	out := URL(in)
	if out != in {
		t.Errorf("URL(%q) = %q, want unchanged (no value slot to redact)", in, out)
	}
}

func TestHeadersRedactsFixedSensitiveNamesAndKeyLogic(t *testing.T) {
	in := map[string][]string{
		"Authorization":       {"Bearer abc123"},
		"Proxy-Authorization": {"Basic def456"},
		"Cookie":              {"session=xyz"},
		"Set-Cookie":          {"session=xyz; HttpOnly"},
		"X-Api-Key":           {"key-value"},
		"X-Access-Token":      {"token-value"},
		"Content-Type":        {"application/json"},
		"X-Request-Id":        {"req-1"},
	}
	got := Headers(in)

	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "Set-Cookie", "X-Api-Key", "X-Access-Token"} {
		for _, v := range got[name] {
			if v != Value {
				t.Errorf("Headers()[%q] = %#v, want all values redacted", name, got[name])
			}
		}
	}
	if got["Content-Type"][0] != "application/json" || got["X-Request-Id"][0] != "req-1" {
		t.Errorf("Headers() redacted a non-sensitive header: %#v", got)
	}

	// Input must not be mutated.
	if in["Authorization"][0] != "Bearer abc123" {
		t.Fatalf("input mutated: %#v", in["Authorization"])
	}
}

func TestHeadersEmptyAndNil(t *testing.T) {
	if got := Headers(nil); got != nil {
		t.Errorf("Headers(nil) = %#v, want nil", got)
	}
	if got := Headers(map[string][]string{}); got == nil || len(got) != 0 {
		t.Errorf("Headers(empty) = %#v, want empty non-nil map", got)
	}
}

func TestPathValuesMasksSecretsCarriedAsPathSegments(t *testing.T) {
	const secret = "live-api-key"

	raw := "https://monitoring.vngcloud.vn/api/v1/apikeys/metric/" + secret
	if got := URL(raw); got != raw {
		t.Fatalf("URL() = %q, want the URL unchanged (this test's premise)", got)
	}
	got := PathValues(raw, []string{secret})
	if strings.Contains(got, secret) || !strings.Contains(got, Value) {
		t.Errorf("PathValues() = %q, want the segment masked", got)
	}
}

func TestPathValuesMasksThePathEscapedForm(t *testing.T) {
	const secret = "key/with slash"
	raw := "https://monitoring.vngcloud.vn/api/v1/apikeys/metric/" + url.PathEscape(secret)
	if got := PathValues(raw, []string{secret}); strings.Contains(got, url.PathEscape(secret)) {
		t.Errorf("PathValues() = %q, want the escaped form masked too", got)
	}
}

func TestPathValuesIgnoresEmptySecrets(t *testing.T) {
	raw := "https://monitoring.vngcloud.vn/api/v1/apikeys/metric/list"
	if got := PathValues(raw, []string{""}); got != raw {
		t.Errorf("PathValues() = %q, want unchanged; an empty secret must not match everywhere", got)
	}
}
