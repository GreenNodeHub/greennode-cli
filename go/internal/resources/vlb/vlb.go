// Package vlb registers shell-completion providers for VLB-owned resources
// (load balancers, certificates) into the shared cli registry, so other
// services can complete them without importing the VLB command package.
//
// Coupling note: this file knows VLB API paths/response shapes (not the VLB
// CLI). It is the single isolated place that does.
package vlb

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
)

func init() {
	for key, fn := range registeredKeys() {
		cli.RegisterResourceCompleter(key, fn)
	}
}

func registeredKeys() map[string]cli.CompFunc {
	return map[string]cli.CompFunc{
		"vlb:loadbalancer": vlbclient.CompleteLoadBalancerIDs,
		"vlb:certificate":  vlbclient.CompleteCertificateIDs,
	}
}
