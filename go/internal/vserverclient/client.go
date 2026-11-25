// Package vserverclient adapts the shared internal/cli plumbing to vServer:
// the endpoint name and the vServer-specific resource completers. The client,
// project-id and output helpers are thin wrappers over internal/cli so vServer
// and other project-scoped services (vlb) cannot drift apart.
package vserverclient

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

// BuildClient creates a GreennodeClient for the vServer endpoint from cobra
// command flags, along with the config its project ID comes from.
func BuildClient(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, error) {
	return cli.NewClientWithConfig(cmd, "vserver")
}

// ProjectID extracts and validates the project ID from config.
func ProjectID(cfg *config.Config) (string, error) {
	return cli.ProjectID(cfg)
}

// Output formats and writes the API result to stdout.
func Output(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return cli.OutputWithConfig(cmd, cfg, data)
}

// OutputWithColumns formats and writes the API result to stdout.
// When the output format is "table", only the specified columns are shown in the given order.
func OutputWithColumns(cmd *cobra.Command, cfg *config.Config, data interface{}, columns []string) error {
	return cli.OutputWithColumns(cmd, cfg, data, columns)
}
