package sshkey

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type fixtureToken struct{}

func (fixtureToken) GetToken() (string, error)     { return "fixture-token", nil }
func (fixtureToken) RefreshToken() (string, error) { return "fixture-token", nil }

func TestPrivateKeysRequireExplicitOutputOptIn(t *testing.T) {
	for _, output := range []string{"json", "table", "text"} {
		for _, show := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", output, show), func(t *testing.T) {
				cmd := &cobra.Command{}
				cmd.Flags().String("output", output, "")
				cmd.Flags().String("query", "", "")
				cmd.Flags().String("color", "never", "")
				cmd.Flags().Bool("show-secret", show, "")
				for _, render := range []func(*cobra.Command, any) error{
					func(c *cobra.Command, v any) error { return outputKeyList(c, nil, v) },
					func(c *cobra.Command, v any) error { return outputKeyMutation(c, nil, v) },
				} {
					data := map[string]any{"name": "fixture-name", "publicKey": "fixture-public", "privateKey": "fixture-secret", "priKey": "fixture-secret-alias", "nested": []any{map[string]any{"token": "fixture-token-value", "pubKey": "fixture-public-nested", "private_key": "fixture-secret-nested"}}}
					out := testutil.CaptureStdout(t, func() {
						if err := render(cmd, data); err != nil {
							t.Fatal(err)
						}
					})
					if !show && (strings.Contains(out, "fixture-secret") || strings.Contains(out, "fixture-token-value")) {
						t.Fatalf("secret output without opt-in: %s", out)
					}
					if output != "table" && !strings.Contains(out, "fixture-public") {
						t.Fatal("public key lost")
					}
					if !show && output != "table" && !strings.Contains(out, "[REDACTED]") {
						t.Fatal("missing redaction marker")
					}
					if show && output == "json" && !strings.Contains(out, "fixture-secret") {
						t.Fatal("explicit JSON output lost secret")
					}
				}
			})
		}
	}
}

func TestKeyResponseNeverLeaksThroughDebugOrErrors(t *testing.T) {
	for _, status := range []int{200, 401, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				fmt.Fprint(w, `{"privateKey":"fixture-private","message":"fixture-unstructured-secret"}`)
			}))
			defer server.Close()
			c := client.NewGreennodeClient(server.URL, fixtureToken{}, time.Second, time.Second, true, true)
			var requestErr error
			stderr := testutil.CaptureStderr(t, func() { _, requestErr = requestKey(c, "GET", "/keys", nil, nil) })
			if calls != 1 {
				t.Fatal("sensitive request retried")
			}
			if strings.Contains(stderr, "fixture-private") || strings.Contains(stderr, "fixture-unstructured-secret") {
				t.Fatal("debug leaked response")
			}
			if status != 200 {
				if requestErr == nil {
					t.Fatal("expected API error")
				}
				if strings.Contains(requestErr.Error(), "fixture-private") || strings.Contains(requestErr.Error(), "fixture-unstructured-secret") {
					t.Fatal("error leaked response")
				}
			} else if requestErr != nil {
				t.Fatal(requestErr)
			}
		})
	}
}
