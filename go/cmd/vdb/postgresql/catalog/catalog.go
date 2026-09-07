// Package catalog holds the read-only lookups a PostgreSQL Cluster is built
// from: datastore versions, flavors, volume types and the vBackup location and
// policy lists.
//
// These are separate endpoints from the relational catalog and the values are NOT
// interchangeable — a cluster flavor ID looks like "pgp-...", a cluster volume
// type like "pgst-...", and the spec says each "can only be compatible with a
// PostgreSQL Cluster". Never feed a relational catalog value into a cluster
// create.
package catalog

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// CatalogCmd is the parent command for the PostgreSQL Cluster lookups.
var CatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Look up PostgreSQL Cluster versions, flavors, volume types and backup options",
	Long: "Read-only lists of what a PostgreSQL Cluster can be built from: datastore " +
		"versions, flavors, volume types, and the backup locations and policies its " +
		"backups are stored under.\n\n" +
		"These values are specific to the cluster product — flavor IDs start with " +
		"'pgp-' and volume type IDs with 'pgst-'. The relational catalog " +
		"('grn vdb relational catalog') lists different, incompatible values.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	pgBase          = "/vdb-postgresql/v1"
	datastorePath   = pgBase + "/cluster/datastore"
	flavorsPath     = pgBase + "/cluster/flavors"
	volumeTypesPath = pgBase + "/cluster/volume-types"
	locationsPath   = pgBase + "/backup/location"
	policiesPath    = pgBase + "/backup/policy"

	// configGroupsPath is relational: the cluster product has no config-group
	// endpoint of its own, so config groups are listed there and filtered to the
	// 'cluster' deploy type, which is what a cluster's --config-id accepts.
	configGroupsPath = "/vdb-relational/v1/configurations"
)

// simpleList describes a lookup that needs no input beyond an optional zone.
type simpleList struct {
	use     string
	short   string
	long    string
	path    string
	byZone  bool // offer --zone-id, sent as ?zoneId=
	columns []string
}

var simpleLists = []simpleList{
	{
		use:   "list-datastores",
		short: "List PostgreSQL Cluster versions",
		long: "List the PostgreSQL versions a cluster can be created with. The 'version' " +
			"value is what --datastore-version expects on 'cluster create'.",
		path:    datastorePath,
		columns: []string{"type", "version", "name", "versionName", "licenseName"},
	},
	{
		use:   "list-flavors",
		short: "List PostgreSQL Cluster flavors",
		long: "List the flavors (vCPU/RAM packages) available to a cluster. The 'id' " +
			"('pgp-...') is what --package-id expects on 'cluster create' and " +
			"'cluster resize'.",
		path:    flavorsPath,
		byZone:  true,
		columns: []string{"id", "name", "vcpus", "ram", "platformType", "backupSize", "status", "locateZoneId"},
	},
	{
		use:   "list-volume-types",
		short: "List PostgreSQL Cluster volume types",
		long: "List the volume types available to a cluster, with their size limits and " +
			"provisioned IOPS. The 'id' ('pgst-...') is what --volume-type-id expects.",
		path:    volumeTypesPath,
		byZone:  true,
		columns: []string{"id", "type", "name", "minVolumeSize", "maxVolumeSize", "iops", "status", "zoneId"},
	},
	{
		use:   "list-backup-locations",
		short: "List backup locations",
		long: "List the vBackup locations a cluster's backups can be stored in. The 'id' " +
			"('bk-des-...') is what --backup-location-id expects on 'cluster create'.",
		path:    locationsPath,
		columns: []string{"id", "name", "type", "product", "status", "isDefault", "numberOfBackupInstances"},
	},
	{
		use:   "list-backup-policies",
		short: "List backup policies",
		long: "List the vBackup policies a cluster's backups can follow. The 'id' " +
			"('bk-pol-...') is what --backup-policy-id expects on 'cluster create'. The " +
			"schedule and retention live in the nested config object — use --output json.",
		path:    policiesPath,
		columns: []string{"id", "name", "product", "isDefault", "backupInstanceCount", "createdAt"},
	},
}

func newSimpleList(spec simpleList) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		Long:  spec.long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := get(cmd, spec.path, zoneQuery(cmd), spec.columns); err != nil {
				return fmt.Errorf("failed to %s: %w", strings.ReplaceAll(spec.use, "-", " "), err)
			}
			return nil
		},
	}
	if spec.byZone {
		cmd.Flags().String("zone-id", "", "Restrict results to one availability zone (e.g. HCM03-1A)")
		// Bound here rather than in a central function: the flag only exists on the
		// specs that ask for it, so this is the one place it is guaranteed to be defined.
		cmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion()) //nolint:errcheck
	}
	return cmd
}

func init() {
	for _, spec := range simpleLists {
		CatalogCmd.AddCommand(newSimpleList(spec))
	}
	CatalogCmd.AddCommand(listConfigGroupsCmd)
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

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

func zoneQuery(cmd *cobra.Command) url.Values {
	query := url.Values{}
	if zoneID, _ := cmd.Flags().GetString("zone-id"); zoneID != "" {
		query.Set("zoneId", zoneID)
	}
	return query
}
