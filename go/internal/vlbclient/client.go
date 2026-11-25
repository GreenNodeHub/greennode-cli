// Package vlbclient adapts the shared internal/cli plumbing to VLB (VNG Load
// Balancer): the endpoint name and the VLB-specific resource completers.
// Mirrors internal/vserverclient — both services are project-scoped and sit on
// the same gateway family.
package vlbclient

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

// ServiceName is the internal/config REGIONS key prefix ("vlb_endpoint").
const ServiceName = "vlb"

// BuildClient creates a GreennodeClient for the VLB endpoint from cobra command
// flags, along with the config its project ID comes from.
func BuildClient(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, error) {
	return cli.NewClientWithConfig(cmd, ServiceName)
}

// ProjectID extracts and validates the project ID from config. Every VLB path
// is /v2/{projectId}/..., so this is required for all commands.
func ProjectID(cfg *config.Config) (string, error) {
	return cli.ProjectID(cfg)
}

// Output formats and writes the API result to stdout.
func Output(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return cli.OutputWithConfig(cmd, cfg, data)
}

// OutputWithColumns formats and writes the API result to stdout. When the output
// format is "table", only the named columns are shown, in the given order.
func OutputWithColumns(cmd *cobra.Command, cfg *config.Config, data interface{}, columns []string) error {
	return cli.OutputWithColumns(cmd, cfg, data, columns)
}
