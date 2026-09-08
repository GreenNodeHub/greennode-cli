package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

// Waiting is not a nicety for VLB: every create/update/delete puts the load
// balancer into UPDATING, and the next mutation is rejected with
// "400 ... is not ready" until it returns to ACTIVE. Any script doing more than
// one thing has to poll, so the CLI provides the poll.

// statusOf reads displayStatus out of a GET response. VLB wraps the resource in
// a "data" envelope and names the field displayStatus — not "status" as VKS
// does — so this cannot share vks's extractor.
func statusOf(result interface{}) string {
	s, _ := unwrapData(result)["displayStatus"].(string)
	return s
}

// terminalFailureStatuses are the states a load balancer will not leave on its
// own, so a waiter should stop rather than poll until timeout.
var terminalFailureStatuses = map[string]bool{"ERROR": true, "FAILED": true}

// evaluateActive decides one poll for wait-active. CREATING, UPDATING and
// DELETING are all transient — keep polling. A 404 is fatal: the load balancer
// the caller named does not exist, and waiting will not conjure it.
func evaluateActive(result interface{}, err error) (done bool, failed bool, status string, fatal error) {
	if err != nil {
		if cli.IsNotFound(err) || cli.IsPermanentAPIError(err) {
			return false, false, "", err
		}
		// Transient (5xx, network) — the client already retried; poll again.
		return false, false, "", nil
	}
	status = statusOf(result)
	switch {
	case status == "ACTIVE":
		return true, false, status, nil
	case terminalFailureStatuses[status]:
		return false, true, status, nil
	default:
		return false, false, status, nil
	}
}

// evaluateDeleted decides one poll for wait-deleted: the resource being gone
// (404) is the success condition.
func evaluateDeleted(result interface{}, err error) (done bool, failed bool, status string, fatal error) {
	if err != nil {
		if cli.IsNotFound(err) {
			return true, false, "DELETED", nil
		}
		if cli.IsPermanentAPIError(err) {
			return false, false, "", err
		}
		return false, false, "", nil
	}
	status = statusOf(result)
	switch {
	case status == "ACTIVE":
		// Back to ACTIVE means the delete never took effect.
		return false, true, status, nil
	case terminalFailureStatuses[status]:
		return false, true, status, nil
	default:
		return false, false, status, nil
	}
}

var waitActiveCmd = &cobra.Command{
	Use:   "wait-active",
	Short: "Wait until a load balancer reaches ACTIVE",
	Long: "Poll until the load balancer is ACTIVE. Use this between mutations: " +
		"every change puts the load balancer into UPDATING, and the next change " +
		"is rejected until it settles.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWait(cmd, evaluateActive, "reach ACTIVE")
	},
}

var waitDeletedCmd = &cobra.Command{
	Use:   "wait-deleted",
	Short: "Wait until a load balancer is fully deleted",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWait(cmd, evaluateDeleted, "be deleted")
	},
}

func init() {
	// Defaults are sized from observed behaviour: provisioning a new load
	// balancer takes a few minutes, an update settles in well under one.
	for _, c := range []*cobra.Command{waitActiveCmd, waitDeletedCmd} {
		addLBFlag(c)
		c.Flags().Int("delay", 10, "Seconds between polls")
		c.Flags().Int("max-attempts", 90, "Maximum poll attempts")
	}
}

func runWait(cmd *cobra.Command, eval cli.WaitEvaluator, goal string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}
	delay, _ := cmd.Flags().GetInt("delay")
	maxAttempts, _ := cmd.Flags().GetInt("max-attempts")
	if delay <= 0 {
		return fmt.Errorf("--delay must be greater than 0, got %d", delay)
	}
	if maxAttempts <= 0 {
		return fmt.Errorf("--max-attempts must be greater than 0, got %d", maxAttempts)
	}

	apiClient, _, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	describe := func() (interface{}, error) {
		return apiClient.Get(vlbclient.LoadBalancer(projectID, lbID), nil)
	}
	return cli.RunWaiter(
		fmt.Sprintf("load balancer %s", lbID),
		fmt.Sprintf("Successfully waited for load balancer %s to %s", lbID, goal),
		describe, eval, delay, maxAttempts)
}
