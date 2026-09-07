package user

import (
	"context"
	"fmt"

	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/cluster"
	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/topic"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completers live here; flags are bound in the init() of the file that DEFINES
// them — see the init-order note in cmd/vdb/CLAUDE.md.

// UserResourceKey lists the user IDs of the cluster named on the command line.
// CONTEXT-DEPENDENT: users are scoped to a cluster, so it yields nothing until
// --cluster-id is set.
const UserResourceKey = "vdb:kafka-user"

func init() {
	cli.RegisterResourceCompleter(UserResourceKey, cli.FlagFromAPI(userIDs))
}

func clusterIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(cluster.ClusterResourceKey)
}

func userIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(UserResourceKey)
}

// topicNameCompletion completes the permission flags. It uses the topic NAME key,
// not the ID one: the permission fields are lists of topic names, and an ID there is
// accepted and then matches nothing.
func topicNameCompletion() cli.CompFunc {
	return cli.ResourceCompletion(topic.TopicNameResourceKey)
}

func userIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if clusterID == "" {
		return nil, nil
	}
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	result, err := apiClient.Get(usersPath(clusterID), nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(vdbclient.Unwrap(result), "id"), nil
}

// usersPath is the user collection of a cluster, built from the cluster package's
// helper rather than by string-joining the prefix here: one owner per path.
func usersPath(clusterID string) string {
	return cluster.ClusterPath(clusterID, "/users")
}

func userPath(clusterID, userID string) string {
	return usersPath(clusterID) + "/" + userID
}

func validateClusterID(clusterID string) error {
	return cluster.ValidateClusterID(clusterID)
}

func validateIDs(clusterID, userID string) error {
	if err := cluster.ValidateClusterID(clusterID); err != nil {
		return err
	}
	return validator.ValidateID(userID, "user-id")
}

// fetchUser reads one user, reporting a null payload as a miss rather than handing
// back zeroed fields.
func fetchUser(apiClient *vdbclient.Client, clusterID, userID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(userPath(clusterID, userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read user %s of Kafka cluster %s: %w", userID, clusterID, err)
	}
	userObj, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("user %s of Kafka cluster %s not found (the API returned an empty payload)",
			userID, clusterID)
	}
	return userObj, nil
}
