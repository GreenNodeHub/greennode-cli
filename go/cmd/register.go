package cmd

// Product services self-register via their package init(). Blank-importing them
// here is the ONLY change needed to mount a new product CLI — root.go iterates
// the registry and never needs editing.
//
// vServer is registered in register_vserver.go behind the "!vks_only" build tag,
// so the public release binary (built with `-tags vks_only`) ships VKS only while
// vServer is still under development.
import (
	_ "github.com/greennodehub/greennode-cli/cmd/vks"
	_ "github.com/greennodehub/greennode-cli/cmd/vserver"
	_ "github.com/greennodehub/greennode-cli/internal/resources/vserver"
)
