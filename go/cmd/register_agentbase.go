package cmd

// GreenNode AgentBase command group. AgentBase is now part of the default grn
// binary and the public release build (-tags vks_only): the build constraint
// was dropped when agentbase became generally available. vServer remains gated
// behind !vks_only (see register_vserver.go) until it is ready to release.
import (
	_ "github.com/greennodehub/greennode-cli/cmd/agentbase"
)
