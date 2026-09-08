package vks

import (
	"errors"
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

// statusOf extracts the "status" string from a decoded JSON object response.
func statusOf(result interface{}) string {
	m, ok := result.(map[string]interface{})
	if !ok {
		return ""
	}
	s, _ := m["status"].(string)
	return s
}

// isPermanentAPIError reports whether err is an HTTP client error that will not
// resolve by polling again. Shared with every other product's waiters.
func isPermanentAPIError(err error) bool {
	return cli.IsPermanentAPIError(err)
}

// evaluateActive decides one poll for an "*-active" waiter. A fatal error aborts
// the wait immediately (permanent client errors / resource not found); other
// errors are treated as transient and polling continues.
func evaluateActive(result interface{}, err error) (done bool, failed bool, status string, fatal error) {
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return false, false, "", err
		}
		if isPermanentAPIError(err) {
			return false, false, "", err
		}
		return false, false, "", nil
	}
	status = statusOf(result)
	switch status {
	case "ACTIVE":
		return true, false, status, nil
	case "ERROR", "FAILED":
		return false, true, status, nil
	default:
		return false, false, status, nil
	}
}

// evaluateDeleted decides one poll for a "*-deleted" waiter. A 404 means the
// resource is gone (success); permanent client errors abort; other errors are
// transient.
func evaluateDeleted(result interface{}, err error) (done bool, failed bool, status string, fatal error) {
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return true, false, "DELETED", nil
		}
		if isPermanentAPIError(err) {
			return false, false, "", err
		}
		return false, false, "", nil
	}
	status = statusOf(result)
	if status == "ACTIVE" {
		return false, true, status, nil
	}
	return false, false, status, nil
}

// waitCmd is the parent for all `grn vks wait <condition>` subcommands.
var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for a VKS resource to reach a desired state",
	Long:  "Poll until a cluster or node group reaches the requested state (active or deleted).",
}

// evaluator decides, for one poll, whether the waiter is done, has failed, or
// hit a fatal error that should abort polling immediately.
type evaluator = cli.WaitEvaluator

// runWaiter polls describe() until eval says the wait is over. The polling,
// progress reporting and exit-code convention live in internal/cli so every
// product behaves identically; only the evaluators differ per service.
func runWaiter(label, successMsg string, describe func() (interface{}, error), eval evaluator, delay, maxAttempts int) error {
	return cli.RunWaiter(label, successMsg, describe, eval, delay, maxAttempts)
}

var (
	waitClusterActiveSub    = &cobra.Command{Use: "cluster-active", Short: "Wait until a cluster reaches ACTIVE", RunE: runWaitClusterActive}
	waitClusterDeletedSub   = &cobra.Command{Use: "cluster-deleted", Short: "Wait until a cluster is deleted", RunE: runWaitClusterDeleted}
	waitNodegroupActiveSub  = &cobra.Command{Use: "nodegroup-active", Short: "Wait until a node group reaches ACTIVE", RunE: runWaitNodegroupActive}
	waitNodegroupDeletedSub = &cobra.Command{Use: "nodegroup-deleted", Short: "Wait until a node group is deleted", RunE: runWaitNodegroupDeleted}
)

func init() {
	for _, c := range []*cobra.Command{waitClusterActiveSub, waitClusterDeletedSub, waitNodegroupActiveSub, waitNodegroupDeletedSub} {
		c.Flags().String("cluster-id", "", "Cluster ID (required)")
		c.Flags().Int("delay", 30, "Seconds between polls")
		c.MarkFlagRequired("cluster-id")
	}
	for _, c := range []*cobra.Command{waitNodegroupActiveSub, waitNodegroupDeletedSub} {
		c.Flags().String("nodegroup-id", "", "Node group ID (required)")
		c.MarkFlagRequired("nodegroup-id")
	}
	waitClusterActiveSub.Flags().Int("max-attempts", 40, "Maximum poll attempts")
	waitClusterDeletedSub.Flags().Int("max-attempts", 40, "Maximum poll attempts")
	waitNodegroupActiveSub.Flags().Int("max-attempts", 80, "Maximum poll attempts")
	waitNodegroupDeletedSub.Flags().Int("max-attempts", 40, "Maximum poll attempts")

	waitCmd.AddCommand(waitClusterActiveSub, waitClusterDeletedSub, waitNodegroupActiveSub, waitNodegroupDeletedSub)
}

func runWaitClusterActive(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	delay, _ := cmd.Flags().GetInt("delay")
	maxAttempts, _ := cmd.Flags().GetInt("max-attempts")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	describe := func() (interface{}, error) {
		return apiClient.Get(fmt.Sprintf("/v1/clusters/%s", clusterID), nil)
	}
	return runWaiter(fmt.Sprintf("cluster %s", clusterID), "Successfully waited for cluster to reach ACTIVE", describe, evaluateActive, delay, maxAttempts)
}

func runWaitClusterDeleted(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	delay, _ := cmd.Flags().GetInt("delay")
	maxAttempts, _ := cmd.Flags().GetInt("max-attempts")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	describe := func() (interface{}, error) {
		return apiClient.Get(fmt.Sprintf("/v1/clusters/%s", clusterID), nil)
	}
	return runWaiter(fmt.Sprintf("cluster %s", clusterID), "Successfully waited for cluster to be deleted", describe, evaluateDeleted, delay, maxAttempts)
}

func runWaitNodegroupActive(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	delay, _ := cmd.Flags().GetInt("delay")
	maxAttempts, _ := cmd.Flags().GetInt("max-attempts")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	describe := func() (interface{}, error) {
		return apiClient.Get(fmt.Sprintf("/v1/clusters/%s/node-groups/%s", clusterID, nodegroupID), nil)
	}
	return runWaiter(fmt.Sprintf("node group %s", nodegroupID), "Successfully waited for node group to reach ACTIVE", describe, evaluateActive, delay, maxAttempts)
}

func runWaitNodegroupDeleted(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	delay, _ := cmd.Flags().GetInt("delay")
	maxAttempts, _ := cmd.Flags().GetInt("max-attempts")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	describe := func() (interface{}, error) {
		return apiClient.Get(fmt.Sprintf("/v1/clusters/%s/node-groups/%s", clusterID, nodegroupID), nil)
	}
	return runWaiter(fmt.Sprintf("node group %s", nodegroupID), "Successfully waited for node group to be deleted", describe, evaluateDeleted, delay, maxAttempts)
}
