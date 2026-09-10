package vlb

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type pathParameter = opengine.PathParam

type queryParameter = opengine.QueryParam

type bodyContract = opengine.BodyContract

const (
	queryInteger = opengine.QueryInteger
	queryID      = opengine.QueryID
	queryObject  = opengine.QueryObject
)

type vlbAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type clientFactory func(*cobra.Command) (vlbAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (vlbAPI, error) {
	return cli.NewClient(cmd, "vlb")
}

var _ vlbAPI = (*client.GreennodeClient)(nil)

func isMutation(op operation) bool {
	return op.Method != http.MethodGet
}

var vlbSpec = opengine.Spec[vlbAPI]{
	ServiceName: "vLB",
	NewClient: func(cmd *cobra.Command, _ opengine.Descriptor) (vlbAPI, error) {
		return newClient(cmd)
	},
	Execute: func(_ *cobra.Command, apiClient vlbAPI, d opengine.Descriptor, path string, query map[string]string, body, _ any) (client.HTTPResponse, error) {
		return apiClient.RequestWithStatus(d.Method, path, query, body)
	},
	ResponseError: func(d opengine.Descriptor, response client.HTTPResponse) error {
		return responseError(response, d)
	},
}

func newOperationCommand(op operation) *cobra.Command {
	op.Mutation = isMutation(op)
	return opengine.NewCommand(vlbSpec, op)
}

func responseError(response client.HTTPResponse, op operation) error {
	if response.StatusCode != op.Status {
		return fmt.Errorf("vLB API returned unexpected HTTP %d for %s %s; expected HTTP %d", response.StatusCode, op.Method, op.Path, op.Status)
	}
	if response.Empty {
		if !op.ResponseBody {
			return nil
		}
		return fmt.Errorf("vLB API returned an empty HTTP %d response for %s %s; expected JSON", response.StatusCode, op.Method, op.Path)
	}

	if !op.ResponseBody {
		return fmt.Errorf("vLB API returned a body for %s %s; expected an empty HTTP %d response", op.Method, op.Path, op.Status)
	}
	envelope, ok := response.Data.(map[string]any)
	if !ok {
		return nil
	}
	success, ok := envelope["success"].(bool)
	if !ok || success {
		return nil
	}
	for _, key := range []string{"errorMsg", "message"} {
		if message, ok := envelope[key].(string); ok && strings.TrimSpace(message) != "" {
			return errors.New(message)
		}
	}
	if code, ok := envelope["code"]; ok && code != nil {
		return fmt.Errorf("vLB API reported failure with code %v", code)
	}
	return errors.New("vLB API reported failure")
}
