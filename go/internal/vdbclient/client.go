// Package vdbclient holds the plumbing shared by every `grn vdb` command group:
// client construction, the vDB response-envelope unwrap, list-query building and
// output helpers.
//
// vDB differs from both vks and vserver in ways that are easy to get wrong — see
// go/cmd/vdb/CLAUDE.md for the full list. The two that shape this package:
//
//   - No project ID anywhere in the URL (unlike vserver), so there is no
//     ProjectID helper here on purpose.
//   - Every response is wrapped in {code, message, data}; list responses nest a
//     second data inside that. Unwrap handles both.
package vdbclient

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

// UserTypeHeader is the vDB header selecting the billing flow on endpoints that
// create or resize chargeable resources: ROOT_USER (Checkout) or IAM_USER
// (Auto Payment). The API defaults to ROOT_USER when it is absent.
const UserTypeHeader = "user-type"

// UserTypeValues are the accepted --user-type values, used for shell completion.
var UserTypeValues = []string{"ROOT_USER", "IAM_USER"}

// Client is the vdb HTTP client: the shared GreennodeClient with vDB's error
// payload unpacked into every error message (see error.go). It exposes the same
// method set, so commands call it exactly like the shared client.
//
// The wrapper exists so the improvement applies to all vdb requests at one place
// rather than at each of the ~30 call sites, and so the shared client's error
// formatting — which vks and vserver depend on — stays untouched.
type Client struct {
	*client.GreennodeClient
}

// Every method passes the request body to enrich, which uses it to keep any secret the
// body carried out of the error message — the API echoes a rejected password back in
// plain text (verified live on 2026-08-13: "Redis password [hunter2...] contains invalid
// character"), and that message goes to the terminal and into whatever captures it.
func (c *Client) Get(path string, params map[string]string) (interface{}, error) {
	result, err := c.GreennodeClient.Get(path, params)
	return result, enrich(err, nil)
}

func (c *Client) Post(path string, body interface{}) (interface{}, error) {
	result, err := c.GreennodeClient.Post(path, body)
	return result, enrich(err, body)
}

func (c *Client) Put(path string, body interface{}) (interface{}, error) {
	result, err := c.GreennodeClient.Put(path, body)
	return result, enrich(err, body)
}

func (c *Client) Patch(path string, body interface{}) (interface{}, error) {
	result, err := c.GreennodeClient.Patch(path, body)
	return result, enrich(err, body)
}

func (c *Client) Delete(path string, params map[string]string) (interface{}, error) {
	result, err := c.GreennodeClient.Delete(path, params)
	return result, enrich(err, nil)
}

// DeleteWithBody is needed because vdb has DELETE endpoints that take a JSON array
// body (relational backup and configuration deletes).
func (c *Client) DeleteWithBody(path string, body interface{}) (interface{}, error) {
	result, err := c.GreennodeClient.DeleteWithBody(path, body)
	return result, enrich(err, body)
}

// Request is Post/Put with query parameters. The Kafka API passes the arguments of
// several mutating operations in the query string of a PUT that has no body at all
// (?count=&rebalance=, ?size=, ?storageType=, ?enable=), which Put cannot express.
func (c *Client) Request(method, path string, params map[string]string, body interface{}) (interface{}, error) {
	result, err := c.GreennodeClient.Request(method, path, params, body)
	return result, enrich(err, body)
}

// NoContent sends a request and DISCARDS the response body, reporting only whether
// it succeeded.
//
// Eleven Kafka operations — every mutating one whose result is not a new object —
// declare their 200 response as a bare `string` in the spec, with no schema for
// what that string contains. The product decision is that those carry no
// information: the HTTP status is the whole result.
//
// Live behaviour, verified 2026-08-17 on topic update (200) and topic delete (204):
// the body is EMPTY. Printing that through the normal path would render "{}" where
// every other command prints a resource, which reads like a result and is not one.
// And because the spec promises a string rather than an object, a future
// non-JSON body would make the JSON parse fail on a call that succeeded — so
// skipping the parse is what keeps the outcome tied to the status.
//
// A command using this prints its own confirmation and points at the read command
// that shows the effect. Errors are enriched exactly as elsewhere.
func (c *Client) NoContent(method, path string, params map[string]string, body interface{}) error {
	_, err := c.GreennodeClient.RequestRaw(method, path, params, body)
	return enrich(err, body)
}

// BuildClient creates a client for the vdb service. It resolves the endpoint via
// the "vdb_endpoint" REGIONS key and attaches the user-type header from
// --user-type when set. Unlike vserverclient.BuildClient it returns no config —
// vdb needs no project ID.
func BuildClient(cmd *cobra.Command) (*Client, error) {
	apiClient, err := cli.NewClient(cmd, "vdb")
	if err != nil {
		return nil, err
	}

	// Empty means "not set": applyHeaders skips empty values, so the API applies
	// its own ROOT_USER default rather than us hard-coding it.
	if userType, err := cmd.Flags().GetString("user-type"); err == nil && userType != "" {
		apiClient.SetHeaders(map[string]string{UserTypeHeader: userType})
	}

	return &Client{GreennodeClient: apiClient}, nil
}
