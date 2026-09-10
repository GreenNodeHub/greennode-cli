package cmd

// Imported services self-register in both default and release builds.
import (
	_ "github.com/greennodehub/greennode-cli/cmd/vbackup"
	_ "github.com/greennodehub/greennode-cli/cmd/vcr"
	_ "github.com/greennodehub/greennode-cli/cmd/vks"
	_ "github.com/greennodehub/greennode-cli/cmd/vserver"
	_ "github.com/greennodehub/greennode-cli/internal/resources/vserver"
)
