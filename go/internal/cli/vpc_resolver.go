package cli

import (
	"sync"

	"github.com/spf13/cobra"
)

// Cluster-to-VPC resolution lives behind this hook so that resource providers
// which only know vserver APIs (internal/resources/vserver) can still complete
// subnets for commands that take --cluster-id but no --vpc-id, without importing
// another service's API paths. The service that owns the cluster API registers
// the resolver.

var (
	vpcResolverMu      sync.RWMutex
	clusterVPCResolver func(cmd *cobra.Command, clusterID string) (string, error)
)

// RegisterClusterVPCResolver registers the resolver used by ResolveClusterVPC.
// Called from the owning service's init(). Concurrency-safe; last writer wins.
func RegisterClusterVPCResolver(fn func(cmd *cobra.Command, clusterID string) (string, error)) {
	vpcResolverMu.Lock()
	defer vpcResolverMu.Unlock()
	clusterVPCResolver = fn
}

// ResolveClusterVPC returns the ID of the VPC owning clusterID. It yields "" with
// no error when no resolver is registered or clusterID is empty, so callers can
// treat "unknown" and "unavailable" the same way.
func ResolveClusterVPC(cmd *cobra.Command, clusterID string) (string, error) {
	vpcResolverMu.RLock()
	fn := clusterVPCResolver
	vpcResolverMu.RUnlock()
	if fn == nil || clusterID == "" {
		return "", nil
	}
	return fn(cmd, clusterID)
}
