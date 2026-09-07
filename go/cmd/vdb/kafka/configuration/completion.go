package configuration

import (
	"context"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completers live here; flags are bound in the init() of the file that DEFINES
// them — see the init-order note in cmd/vdb/CLAUDE.md.

const (
	// ConfigGroupResourceKey lists Kafka config GROUP ids.
	ConfigGroupResourceKey = "vdb:kafka-config-group"

	// ConfigGroupVersionResourceKey lists config group VERSION ids — the values a
	// cluster is attached to. It is a separate key because the two are not
	// interchangeable: 'cluster update-config-group' takes a version and rejects a
	// group. Kafka has no endpoint listing versions across groups, so the completer
	// reads the group listing and collects the versions nested in each row.
	ConfigGroupVersionResourceKey = "vdb:kafka-config-group-version"
)

func init() {
	cli.RegisterResourceCompleter(ConfigGroupResourceKey, cli.FlagFromAPI(configGroupIDs))
	cli.RegisterResourceCompleter(ConfigGroupVersionResourceKey, cli.FlagFromAPI(configGroupVersionIDs))
}

func configGroupIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(ConfigGroupResourceKey)
}

func configGroupVersionIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(ConfigGroupVersionResourceKey)
}

func listConfigGroups(cmd *cobra.Command) ([]interface{}, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	result, err := apiClient.Get(basePath, nil)
	if err != nil {
		return nil, err
	}
	groups, _ := vdbclient.Unwrap(result).([]interface{})
	return groups, nil
}

func configGroupIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	groups, err := listConfigGroups(cmd)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(groups, "id"), nil
}

// configGroupVersionIDs digs one level into each group's nested versions array. When
// --config-group-id is on the command line it narrows to that group, which is what
// 'configuration get-version' wants; otherwise it offers every version, which is what
// 'cluster update-config-group' wants.
func configGroupVersionIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	groups, err := listConfigGroups(cmd)
	if err != nil {
		return nil, err
	}
	wanted, _ := cmd.Flags().GetString("config-group-id")

	var out []string
	for _, item := range groups {
		group, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if wanted != "" {
			if id, _ := group["id"].(string); id != wanted {
				continue
			}
		}
		versions, _ := group["versions"].([]interface{})
		out = append(out, cli.ExtractIDs(versions, "id")...)
	}
	return out, nil
}

func validateConfigGroupID(configGroupID string) error {
	return validator.ValidateID(configGroupID, "config-group-id")
}

func validateVersionID(versionID string) error {
	return validator.ValidateID(versionID, "config-group-version-id")
}
