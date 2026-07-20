//go:build agentbase

package cmd

// GreenNode AgentBase command group. Compiled in ONLY when the build is invoked
// with `-tags agentbase`. The default grn binary and the public release build
// (which uses `-tags vks_only`) both exclude it while agentbase is still under
// development. Drop this build constraint once agentbase is feature-complete
// and ready to ship in the default binary.
import (
	_ "github.com/vngcloud/greennode-cli/cmd/agentbase"
)
