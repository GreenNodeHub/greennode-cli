// Package catalog holds the Kafka read-only lookups.
//
// They live under `/vdb-kafka/database/*`, which looks like MemoryStore's
// `/vdb-memory/v1/database/*` but offers a different set: Kafka has no engine,
// datastore, network, subnet or config-group lookup here. Versions come from the
// flavor listing, networks and subnets from the RELATIONAL catalog (they are
// project-wide), and config groups have their own noun because they are versioned.
//
// `GET /vdb-kafka/database/configs` is deliberately not exposed as a command: it
// returns the service's own application settings as a map<string,string> — regexes,
// per-user quotas, forced broker properties — which is not something a CLI user acts
// on. It IS read internally, because it is the only place the Kafka version list
// exists; see KafkaVersionResourceKey in completion.go.
package catalog

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// CatalogCmd is the parent command for the Kafka lookups.
var CatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Look up Kafka versions, flavors and volume types",
	Long: "Read-only lists of what a Kafka cluster can be built from: broker flavors, " +
		"the instance families they belong to, CPU platform codes and volume types.\n\n" +
		"Kafka VERSIONS come from 'list-flavors' — there is no datastore endpoint here, " +
		"unlike the other vDB products.\n\n" +
		"Networks and subnets are NOT here either: Kafka has no endpoint for them, so " +
		"use 'grn vdb relational catalog list-networks' and 'list-subnets' — they are " +
		"project-wide.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath = "/vdb-kafka/database"

	familiesPath    = basePath + "/families"
	codesPath       = basePath + "/codes"
	flavorsPath     = basePath + "/flavors"
	volumeTypesPath = basePath + "/volume-types"
)

// volumeTypeItemsKey is the second level the volume-type listing hides its items
// under. It is the one Kafka catalog endpoint that does NOT put a bare array in the
// envelope: `data` is {projectId, data: [...]}, so the items are one level further
// down. Stripping it here means --output json shows the volume types rather than a
// wrapper, and the completer reads them at all.
const volumeTypeItemsKey = "data"

// simpleList describes a lookup that takes no input at all. Kafka's catalog
// endpoints have no zone parameter, unlike relational's and memorystore's.
type simpleList struct {
	use      string
	short    string
	long     string
	path     string
	itemsKey string
	columns  []string
}

var simpleLists = []simpleList{
	{
		use:     "list-families",
		short:   "List instance families",
		long:    "List the instance families (CPU/memory profiles) that broker flavors belong to.",
		path:    familiesPath,
		columns: []string{"group", "key", "value", "name", "description"},
	},
	{
		use:     "list-flavor-codes",
		short:   "List CPU platform codes",
		long:    "List the CPU platform codes that broker flavors are grouped by.",
		path:    codesPath,
		columns: []string{"key", "value", "familyType", "description"},
	},
	{
		use:   "list-volume-types",
		short: "List volume types for Kafka brokers",
		long: "List the volume types a Kafka broker can use, with the size range each " +
			"one allows.\n\n" +
			"'cluster create --volume-type' and 'cluster update-volume-type' take the " +
			"identifier from here. These values belong to Kafka alone — a volume type " +
			"name from Relational Database or PostgreSQL Cluster is rejected.",
		path:     volumeTypesPath,
		itemsKey: volumeTypeItemsKey,
		columns:  []string{"kafkaUuid", "type", "displayName", "minVolumeSize", "maxVolumeSize", "iops", "zoneId"},
	},
}

func newSimpleList(spec simpleList) *cobra.Command {
	return &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		Long:  spec.long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := get(cmd, spec.path, spec.itemsKey, spec.columns); err != nil {
				return fmt.Errorf("failed to %s: %w", strings.ReplaceAll(spec.use, "-", " "), err)
			}
			return nil
		},
	}
}

func init() {
	for _, spec := range simpleLists {
		CatalogCmd.AddCommand(newSimpleList(spec))
	}
	CatalogCmd.AddCommand(listFlavorsCmd)
}

// get issues the GET and prints it with a column set. These four are among the 9
// Kafka endpoints that DO use the {code, message, data} envelope, unlike the
// cluster/topic/user ones — so Unwrap actually strips something here.
func get(cmd *cobra.Command, path, itemsKey string, columns []string) error {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(path, nil)
	if err != nil {
		return err
	}

	return vdbclient.OutputWithColumns(cmd, items(vdbclient.Unwrap(result), itemsKey), columns)
}

// items digs one level further into a payload that wraps its list under a key. With
// an empty key it is the identity, and it leaves a payload that does not carry the
// key alone rather than returning nothing — so a backend that flattens the shape
// later does not break the command.
func items(payload interface{}, key string) interface{} {
	if key == "" {
		return payload
	}
	object, ok := payload.(map[string]interface{})
	if !ok {
		return payload
	}
	nested, ok := object[key]
	if !ok {
		return payload
	}
	return nested
}
