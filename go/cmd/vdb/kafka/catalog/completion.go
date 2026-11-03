package catalog

import (
	"context"
	"encoding/json"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completion keys owned by these endpoints. They are distinct from every other
// product's even where the names look alike — a Kafka flavor is not a relational one
// and a Kafka volume type is not a PostgreSQL Cluster one.
const (
	// KafkaVersionResourceKey lists the Kafka versions on offer.
	//
	// They come from GET /vdb-kafka/database/configs, which is the ONLY place in the
	// API that carries them: the flavor listing has no version field, and Kafka has no
	// datastore endpoint. That path deliberately has no command — it returns the
	// service's own settings, which is not something a user acts on — but it is the
	// right source for this one value. Verified live 2026-08-17: kafkaVersions is
	// ["3.6.0","3.6.1","3.7.0"].
	KafkaVersionResourceKey = "vdb:kafka-version"

	// FlavorResourceKey lists broker flavors for --flavor-id.
	//
	// The request field `serverFlavorId` takes the flavor's `flavorId` — the opaque
	// "flav-…" string, NOT the numeric `id` that Relational Database's --package-id
	// uses. Confirmed live 2026-08-17 against a running cluster, whose serverFlavorId
	// matched a flavorId in this listing. `id` stays as a fallback in case a listing
	// omits the string.
	FlavorResourceKey = "vdb:kafka-flavor"

	// VolumeTypeResourceKey lists volume types for --volume-type.
	//
	// The request field `kafkaStorageType` takes `kafkaUuid` — the "vtype-…" string
	// that exists on VolumeTypeInfo for this product specifically — NOT the display
	// `type` ("kafka.Gen2-NVMe2-IOPS3000") and not the numeric `id`. Confirmed live
	// 2026-08-17 the same way. This is the opposite of Relational Database, where the
	// volume type is passed by NAME.
	VolumeTypeResourceKey = "vdb:kafka-volume-type"
)

// configsPath is the service-settings endpoint. It is read here for the Kafka version
// list and nowhere else, and it intentionally has no command — see
// KafkaVersionResourceKey and the note in catalog.go.
const configsPath = basePath + "/configs"

// kafkaVersionsKey is the entry holding the versions. Its value is a JSON array
// encoded INSIDE a string ("[\"3.6.0\",…]"), because the payload is a flat
// map<string,string> — hence the second decode in kafkaVersions.
const kafkaVersionsKey = "kafkaVersions"

// datastoreTypeValues is the one hard-coded completion in the vDB tree.
//
// The repo rule is to complete from the API rather than a literal, but there is no
// endpoint here to complete from: Kafka has no engine or datastore listing. The value
// is verified rather than assumed — live 2026-08-17, `?type=kafka` and `?type=Kafka`
// both return all 24 flavors while `?type=bogus` returns none, so the filter is real,
// case-insensitive, and has exactly this one value.
var datastoreTypeValues = []string{"kafka"}

func init() {
	cli.RegisterResourceCompleter(KafkaVersionResourceKey, cli.FlagFromAPI(kafkaVersions))
	cli.RegisterResourceCompleter(FlavorResourceKey, cli.FlagFromAPI(preferredField(flavorsPath, "", "flavorId", "id")))
	// The volume-type payload hides its items one level down — see volumeTypeItemsKey.
	cli.RegisterResourceCompleter(VolumeTypeResourceKey,
		cli.FlagFromAPI(preferredField(volumeTypesPath, volumeTypeItemsKey, "kafkaUuid", "id")))
}

func kafkaVersionCompletion() cli.CompFunc {
	return cli.ResourceCompletion(KafkaVersionResourceKey)
}

func datastoreTypeCompletion() cli.CompFunc {
	return cli.FlagValues(datastoreTypeValues...)
}

// kafkaVersions reads the version list out of the service settings map.
func kafkaVersions(_ context.Context, cmd *cobra.Command) ([]string, error) {
	payload, err := listing(cmd, configsPath, "")
	if err != nil {
		return nil, err
	}
	settings, ok := payload.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	encoded, _ := settings[kafkaVersionsKey].(string)
	if encoded == "" {
		return nil, nil
	}

	var versions []string
	if err := json.Unmarshal([]byte(encoded), &versions); err != nil {
		// A settings value that stops being a JSON array is not worth failing a
		// completion over; offer nothing instead.
		return nil, nil //nolint:nilerr
	}
	return versions, nil
}

// preferredField reads `field`, falling back to `fallback` when the listing has no
// values under the first, so a backend that stops populating one of the pair leaves
// completion working rather than quietly empty.
func preferredField(path, itemsKey, field, fallback string) func(context.Context, *cobra.Command) ([]string, error) {
	return func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		payload, err := listing(cmd, path, itemsKey)
		if err != nil {
			return nil, err
		}
		if values := vdbclient.ExtractIDValues(payload, field); len(values) > 0 {
			return values, nil
		}
		return vdbclient.ExtractIDValues(payload, fallback), nil
	}
}

// listing fetches a catalog payload. ExtractIDValues is used on the result rather
// than cli.ExtractIDs because several Kafka catalog ids arrive as JSON numbers, which
// the string-only helper skips, leaving completion silently empty.
func listing(cmd *cobra.Command, path, itemsKey string) (interface{}, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	result, err := apiClient.Get(path, nil)
	if err != nil {
		return nil, err
	}
	return items(vdbclient.Unwrap(result), itemsKey), nil
}
