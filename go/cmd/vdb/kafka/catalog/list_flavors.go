package catalog

import (
	"fmt"
	"net/url"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// flavorColumns leads with flavorId because that is what 'cluster create --flavor-id'
// takes — the opaque "flav-…" string, not the numeric `id`, which is kept next to it
// only so the two are not confused. See FlavorResourceKey in completion.go.
var flavorColumns = []string{
	"flavorId", "id", "name", "vcpus", "ram", "priceKey", "description", "locateZoneId",
}

var listFlavorsCmd = &cobra.Command{
	Use:   "list-flavors",
	Short: "List broker flavors, optionally filtered by Kafka version",
	Long: "List the flavors a Kafka broker can run on.\n\n" +
		"'cluster create --flavor-id' takes the flavorId column — the 'flav-...' string, " +
		"NOT the numeric id that Relational Database's --package-id uses.\n\n" +
		"--datastore-type filters (only 'kafka' matches anything). --datastore-version " +
		"is accepted by the API and currently has NO effect: the same 24 flavors come " +
		"back for a real version and for a made-up one, and the rows carry no version " +
		"field to filter on client-side (verified live 2026-08-17). Shell completion for " +
		"it still offers the real versions.\n\n" +
		"Unlike the relational and memorystore catalogs, this endpoint takes NO zone " +
		"parameter — flavors are not listed per zone here.",
	Args: cobra.NoArgs,
	RunE: runListFlavors,
}

func init() {
	f := listFlavorsCmd.Flags()
	f.String("datastore-type", "", "Filter by engine type (the API's 'type' parameter)")
	f.String("datastore-version", "", "Filter by Kafka version")
	// Bound here, next to the flags — see the init-order note in completion.go.
	listFlavorsCmd.RegisterFlagCompletionFunc("datastore-version", kafkaVersionCompletion()) //nolint:errcheck
	listFlavorsCmd.RegisterFlagCompletionFunc("datastore-type", datastoreTypeCompletion())   //nolint:errcheck
}

func runListFlavors(cmd *cobra.Command, args []string) error {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	query := url.Values{}
	if datastoreType, _ := cmd.Flags().GetString("datastore-type"); datastoreType != "" {
		query.Set("type", datastoreType)
	}
	if version, _ := cmd.Flags().GetString("datastore-version"); version != "" {
		query.Set("version", version)
	}

	result, err := vdbclient.Get(apiClient, flavorsPath, query)
	if err != nil {
		return fmt.Errorf("failed to list Kafka flavors: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, flavorColumns)
}
