package vcr

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type pathParameter = opengine.PathParam

type queryParameter = opengine.QueryParam

type bodyContract = opengine.BodyContract

const queryInteger = opengine.QueryInteger

type bodyBinding struct {
	Field string
	Flag  string
}

type vcrAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetry(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetrySensitive(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetrySensitiveRaw(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type clientFactory func(*cobra.Command) (vcrAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (vcrAPI, error) {
	return cli.NewClientWithEndpoint(cmd, endpoint)
}

var _ vcrAPI = (*client.GreennodeClient)(nil)

var vcrSpec = opengine.Spec[vcrAPI]{
	ServiceName: "vCR",
	NewClient: func(cmd *cobra.Command, _ opengine.Descriptor) (vcrAPI, error) {
		return newClient(cmd)
	},
	PostParse: func(cmd *cobra.Command, d opengine.Descriptor, _ string, _ map[string]string, body any) (any, any, error) {
		bindings, _ := d.Extra.([]bodyBinding)
		if err := validateBodyBindings(cmd, body, bindings); err != nil {
			return nil, nil, err
		}
		return body, nil, nil
	},
	Execute: func(_ *cobra.Command, apiClient vcrAPI, d opengine.Descriptor, path string, query map[string]string, body, _ any) (client.HTTPResponse, error) {
		switch {
		case d.SecretResponse && d.RawResponse:
			return apiClient.RequestWithStatusNoRetrySensitiveRaw(d.Method, path, query, body)
		case d.SecretResponse:
			return apiClient.RequestWithStatusNoRetrySensitive(d.Method, path, query, body)
		case d.Mutation:
			return apiClient.RequestWithStatusNoRetry(d.Method, path, query, body)
		default:
			return apiClient.RequestWithStatus(d.Method, path, query, body)
		}
	},
	ResponseError: func(d opengine.Descriptor, response client.HTTPResponse) error {
		return responseError(d, response)
	},
	ShowSecretUsage: "Print returned secret material in command output",
	TransformOutput: func(cmd *cobra.Command, d opengine.Descriptor, data any) (any, error) {
		if !d.SecretResponse {
			return data, nil
		}
		showSecret, _ := cmd.Flags().GetBool("show-secret")
		if showSecret {
			return data, nil
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Secret response redacted. Re-run with --show-secret only when you are ready to handle the credential securely.")
		return redactSecretResponse(data), nil
	},
}

func newOperationCommand(op operation) *cobra.Command {
	return opengine.NewCommand(vcrSpec, op)
}

func validateBodyBindings(cmd *cobra.Command, body any, bindings []bodyBinding) error {
	object, _ := body.(map[string]any)
	for _, binding := range bindings {
		value, ok := object[binding.Field]
		if !ok {
			continue
		}
		bodyID, ok := value.(string)
		if !ok {
			return fmt.Errorf("body field %q must be a string matching --%s", binding.Field, binding.Flag)
		}
		flagID, _ := cmd.Flags().GetString(binding.Flag)
		if bodyID != flagID {
			return fmt.Errorf("body field %q must match --%s", binding.Field, binding.Flag)
		}
	}
	return nil
}

func responseError(op operation, response client.HTTPResponse) error {
	if response.StatusCode != op.Status {
		return fmt.Errorf("vCR API returned HTTP %d for %s %s; expected HTTP %d", response.StatusCode, op.Method, op.Path, op.Status)
	}
	if response.Empty == op.ResponseBody {
		if op.ResponseBody {
			return fmt.Errorf("vCR API returned an empty HTTP %d response for %s %s; expected JSON", response.StatusCode, op.Method, op.Path)
		}
		return fmt.Errorf("vCR API returned a JSON body for %s %s; the documented HTTP %d response is empty", op.Method, op.Path, op.Status)
	}
	return nil
}

func redactSecretResponse(value any) any {
	if _, ok := value.(string); ok {
		return client.RedactedValue
	}
	return client.RedactJSON(value)
}
