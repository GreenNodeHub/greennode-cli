package vbackup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type pathParameter = opengine.PathParam

type queryParameter = opengine.QueryParam

type responseKind uint8

const responseArray responseKind = 1

type vbackupAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type clientFactory func(*cobra.Command) (vbackupAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (vbackupAPI, error) {
	return cli.NewClient(cmd, "vbackup")
}

var _ vbackupAPI = (*client.GreennodeClient)(nil)

// Request schemas are objects with no required properties.
var requiredObjectBody = &opengine.BodyContract{Kind: opengine.ObjectBody, Usage: "Request body as a JSON object"}

var vbackupSpec = opengine.Spec[vbackupAPI]{
	ServiceName: "vBackup Gateway",
	NewClient: func(cmd *cobra.Command, _ opengine.Descriptor) (vbackupAPI, error) {
		return newClient(cmd)
	},
	OfflineValidate: func(cmd *cobra.Command, _ opengine.Descriptor) error {
		return validateExplicitHCM3(cmd)
	},
	LiveValidate: func(cmd *cobra.Command, _ opengine.Descriptor) error {
		return validateLiveHCM3(cmd)
	},
	Execute: func(_ *cobra.Command, apiClient vbackupAPI, d opengine.Descriptor, path string, query map[string]string, body, _ any) (client.HTTPResponse, error) {
		return apiClient.RequestWithStatus(d.Method, path, query, body)
	},
	ResponseError: func(d opengine.Descriptor, response client.HTTPResponse) error {
		return responseError(d, response)
	},
}

func newOperationCommand(op operation) *cobra.Command {
	return opengine.NewCommand(vbackupSpec, op)
}

func responseError(op operation, response client.HTTPResponse) error {
	if response.StatusCode != op.Status {
		return fmt.Errorf("vBackup Gateway returned HTTP %d for %s %s; expected HTTP %d", response.StatusCode, op.Method, op.Path, op.Status)
	}
	if err := opengine.EmptyResponseError("vBackup Gateway", op, response); err != nil {
		return err
	}
	if !op.ResponseBody {
		return nil
	}
	if op.Extra == responseArray {
		if _, ok := response.Data.([]any); !ok {
			return fmt.Errorf("vBackup Gateway expected a JSON array for %s %s", op.Method, op.Path)
		}
	} else if _, ok := response.Data.(map[string]any); !ok {
		return fmt.Errorf("vBackup Gateway expected a JSON object for %s %s", op.Method, op.Path)
	}
	return nil
}

// validateExplicitHCM3 checks flags without loading a profile.
func validateExplicitHCM3(cmd *cobra.Command) error {
	if endpoint, _ := cmd.Flags().GetString("endpoint-url"); endpoint != "" {
		return nil
	}
	region, _ := cmd.Flags().GetString("region")
	if region != "" && region != "HCM-3" {
		return fmt.Errorf("vBackup Gateway commands require region HCM-3, got %q", region)
	}
	return nil
}

// validateLiveHCM3 checks the resolved region before client construction.
func validateLiveHCM3(cmd *cobra.Command) error {
	if endpoint, _ := cmd.Flags().GetString("endpoint-url"); endpoint != "" {
		return nil
	}
	region, _ := cmd.Flags().GetString("region")
	if region == "" {
		profile, _ := cmd.Flags().GetString("profile")
		cfg, err := config.LoadConfig(profile)
		if err != nil {
			return err
		}
		region = cfg.Region
	}
	if region != "HCM-3" {
		return fmt.Errorf("vBackup Gateway commands require region HCM-3, got %q", region)
	}
	return nil
}
