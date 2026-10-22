package vmonitor

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type queryParameter = opengine.QueryParam

type vmonitorAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetrySensitive(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type clientFactory func(*cobra.Command) (vmonitorAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (vmonitorAPI, error) {
	return cli.NewClientWithEndpoint(cmd, endpoint)
}

var _ vmonitorAPI = (*client.GreennodeClient)(nil)

var _ interface{ MaskPathValues([]string) } = (*client.GreennodeClient)(nil)

var requiredObjectBody = &opengine.BodyContract{Kind: opengine.ObjectBody, Usage: "Request body as a JSON object"}

var vmonitorSpec = opengine.Spec[vmonitorAPI]{
	ServiceName: "vMonitor",
	NewClient: func(cmd *cobra.Command, _ opengine.Descriptor) (vmonitorAPI, error) {
		return newClient(cmd)
	},
	Execute: func(_ *cobra.Command, apiClient vmonitorAPI, d opengine.Descriptor, path string, query map[string]string, body, _ any) (client.HTTPResponse, error) {
		if d.SecretResponse {
			return apiClient.RequestWithStatusNoRetrySensitive(d.Method, path, query, body)
		}
		return apiClient.RequestWithStatus(d.Method, path, query, body)
	},
	TransformOutput: func(cmd *cobra.Command, d opengine.Descriptor, data any) (any, error) {
		if !d.SecretResponse {
			return data, nil
		}
		showSecret, _ := cmd.Flags().GetBool("show-secret")
		if showSecret {
			return data, nil
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Credential response redacted. Re-run with --show-secret only when you are ready to handle it securely.")
		return cli.RedactJSON(data), nil
	},
}

func newOperationCommand(op operation) *cobra.Command {
	names, err := pathParameters(op.Path)
	if err != nil {
		panic(fmt.Sprintf("invalid vMonitor operation path %q: %v", op.Path, err))
	}
	op.Paths = make([]opengine.PathParam, len(names))
	for i, name := range names {
		op.Paths[i] = opengine.PathParam{Placeholder: name, Flag: flagName(name), Usage: "Path parameter " + name, Secret: slices.Contains(op.SecretPathParams, name)}
	}
	op.Short = op.Method + " " + op.Path
	return opengine.NewCommand(vmonitorSpec, op)
}

func pathParameters(path string) ([]string, error) {
	var parameters []string
	remaining := path
	for remaining != "" {
		start := strings.IndexByte(remaining, '{')
		if start < 0 {
			if strings.Contains(remaining, "}") {
				return nil, errors.New("unmatched closing brace")
			}
			break
		}
		if strings.Contains(remaining[:start], "}") {
			return nil, errors.New("unmatched closing brace")
		}
		end := strings.IndexByte(remaining[start+1:], '}')
		if end < 0 {
			return nil, errors.New("unclosed path parameter")
		}
		end += start + 1
		name := remaining[start+1 : end]
		if name == "" || strings.ContainsAny(name, "{}") {
			return nil, errors.New("invalid path parameter")
		}
		parameters = append(parameters, name)
		remaining = remaining[end+1:]
	}
	return parameters, nil
}

func flagName(name string) string {
	return opengine.FlagName(name)
}
