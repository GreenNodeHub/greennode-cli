package vlbclient

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

// Resolve builds the client and resolves the project ID in one step — every VLB
// command needs both, and every VLB path embeds the project ID.
func Resolve(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, string, error) {
	apiClient, cfg, err := BuildClient(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	projectID, err := ProjectID(cfg)
	if err != nil {
		return nil, nil, "", err
	}
	return apiClient, cfg, projectID, nil
}

// URL path builders. Keeping them here means every interpolation of a
// caller-supplied ID into a path is in one auditable place; callers must have
// run validator.ValidateID on those IDs first.

// LoadBalancers is the load balancer collection.
func LoadBalancers(projectID string) string {
	return fmt.Sprintf("/v2/%s/loadBalancers", projectID)
}

// LoadBalancer is a single load balancer.
func LoadBalancer(projectID, lbID string) string {
	return fmt.Sprintf("/v2/%s/loadBalancers/%s", projectID, lbID)
}

// LoadBalancerSub is a sub-resource of a load balancer (resize, clone, ...).
func LoadBalancerSub(projectID, lbID, sub string) string {
	return fmt.Sprintf("/v2/%s/loadBalancers/%s/%s", projectID, lbID, sub)
}

// Packages lists the load balancer packages available to the project.
func Packages(projectID string) string {
	return LoadBalancers(projectID) + "/packages"
}

// AllPackages lists every load balancer package, including unavailable ones.
func AllPackages(projectID string) string {
	return LoadBalancers(projectID) + "/all_packages"
}

// Headers lists the HTTP headers a listener can insert.
func Headers(projectID string) string {
	return LoadBalancers(projectID) + "/headers"
}

// LoadBalancersBySubnet lists the load balancers in one subnet.
func LoadBalancersBySubnet(projectID, subnetID string) string {
	return fmt.Sprintf("%s/subnet/%s", LoadBalancers(projectID), subnetID)
}

// Listeners is the listener collection of a load balancer.
func Listeners(projectID, lbID string) string {
	return fmt.Sprintf("/v2/%s/loadBalancers/%s/listeners", projectID, lbID)
}

// Listener is a single listener.
func Listener(projectID, lbID, listenerID string) string {
	return fmt.Sprintf("%s/%s", Listeners(projectID, lbID), listenerID)
}

// Policies is the L7 policy collection of a listener.
func Policies(projectID, lbID, listenerID string) string {
	return fmt.Sprintf("%s/l7policies", Listener(projectID, lbID, listenerID))
}

// Policy is a single L7 policy.
func Policy(projectID, lbID, listenerID, policyID string) string {
	return fmt.Sprintf("%s/%s", Policies(projectID, lbID, listenerID), policyID)
}

// Pools is the pool collection of a load balancer.
func Pools(projectID, lbID string) string {
	return fmt.Sprintf("/v2/%s/loadBalancers/%s/pools", projectID, lbID)
}

// Pool is a single pool.
func Pool(projectID, lbID, poolID string) string {
	return fmt.Sprintf("%s/%s", Pools(projectID, lbID), poolID)
}

// Members is a pool's member collection.
func Members(projectID, lbID, poolID string) string {
	return fmt.Sprintf("%s/members", Pool(projectID, lbID, poolID))
}

// HealthMonitor is a pool's health monitor.
func HealthMonitor(projectID, lbID, poolID string) string {
	return fmt.Sprintf("%s/healthMonitor", Pool(projectID, lbID, poolID))
}

// Certificates is the certificate (CA) collection.
func Certificates(projectID string) string {
	return fmt.Sprintf("/v2/%s/cas", projectID)
}

// Certificate is a single certificate.
func Certificate(projectID, caID string) string {
	return fmt.Sprintf("%s/%s", Certificates(projectID), caID)
}
