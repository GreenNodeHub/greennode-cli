// Package catalog holds the read-only lookup endpoints of the Relational
// Database API — engines, datastore versions, instance families, flavors, zones,
// networks and volume types.
//
// They are grouped under one noun because they share a shape (GET, no filters
// beyond an optional zone, a bare array in the envelope) and one purpose: they
// are what a user consults before creating an instance, and what shell
// completion reads to suggest values for --datastore-type, --datastore-version
// and --zone-id. vDB declares no enums anywhere in its spec, so these endpoints
// are the only source of valid values.
package catalog

import (
	"net/url"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// basePath — every relational catalog endpoint hangs off the instance collection
// rather than a catalog prefix of its own.
const basePath = "/vdb-relational/v1/database-instances"

const (
	enginePath      = basePath + "/engine"
	datastorePath   = basePath + "/datastore"
	familiesPath    = basePath + "/families"
	flavorCodesPath = basePath + "/flavor_zones/codes"
	flavorsPath     = basePath + "/flavors"
	zonesPath       = basePath + "/zones"
	networksPath    = basePath + "/networks"
	subnetsPath     = basePath + "/networks/subnets"
	volumeTypesPath = basePath + "/volume/types"
)

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

// get issues the GET and prints it with a column set. Every catalog endpoint
// returns a single array inside the envelope (volume types wrap theirs in a
// one-key object), so table extraction is unambiguous — unlike instance detail.
func get(cmd *cobra.Command, path string, query url.Values, columns []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := vdbclient.Get(apiClient, path, query)
	if err != nil {
		return err
	}

	return vdbclient.OutputWithColumns(cmd, result, columns)
}

// zoneQuery renders the optional --zone-id filter shared by the flavor, subnet
// and volume-type listings.
func zoneQuery(cmd *cobra.Command) url.Values {
	query := url.Values{}
	if zoneID, _ := cmd.Flags().GetString("zone-id"); zoneID != "" {
		query.Set("zoneId", zoneID)
	}
	return query
}
