package vstoragegateway

import (
	"fmt"
	"net/http"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type queryParameter = opengine.QueryParam

type gatewayAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type clientFactory func(*cobra.Command) (gatewayAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (gatewayAPI, error) {
	return cli.NewClientWithEndpoint(cmd, endpoint)
}

var _ gatewayAPI = (*client.GreennodeClient)(nil)

func allOperations() []operation {
	return []operation{
		{Parent: "region", Use: "list", Short: "List vStorage regions", Method: http.MethodGet, Path: "/v1/regions"},
		{Parent: "project", Use: "list", Short: "List vStorage projects in a region", Method: http.MethodGet, Path: "/v1/projects", Queries: []queryParameter{{WireName: "region_id", Flag: "region-id", Usage: "Query parameter region_id", Required: true, Kind: opengine.QueryID}}},
		{Parent: "container", Use: "list", Short: "List vStorage containers in a project", Method: http.MethodGet, Path: "/v1/containers", Queries: []queryParameter{{WireName: "region_id", Flag: "region-id", Usage: "Query parameter region_id", Required: true, Kind: opengine.QueryID}, {WireName: "project_id", Flag: "project-id", Usage: "Query parameter project_id", Required: true, Kind: opengine.QueryID}}},
	}
}

var vstorageGatewaySpec = opengine.Spec[gatewayAPI]{
	ServiceName: "vMonitor-vStorage Gateway",
	NewClient: func(cmd *cobra.Command, _ opengine.Descriptor) (gatewayAPI, error) {
		return newClient(cmd)
	},
	Execute: func(_ *cobra.Command, apiClient gatewayAPI, d opengine.Descriptor, path string, query map[string]string, _, _ any) (client.HTTPResponse, error) {
		return apiClient.RequestWithStatus(d.Method, path, query, nil)
	},
	ResponseError: func(d opengine.Descriptor, response client.HTTPResponse) error {
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("vStorage Gateway returned HTTP %d; expected 200", response.StatusCode)
		}
		if response.Empty {
			return fmt.Errorf("vMonitor-vStorage Gateway returned an empty HTTP %d response for %s %s; expected JSON", response.StatusCode, d.Method, d.Path)
		}
		if _, ok := response.Data.([]any); !ok {
			return fmt.Errorf("vMonitor-vStorage Gateway returned a non-array JSON response for %s %s; expected an array", d.Method, d.Path)
		}
		return nil
	},
}

func newOperationCommand(op operation) *cobra.Command {
	return opengine.NewCommand(vstorageGatewaySpec, op)
}
