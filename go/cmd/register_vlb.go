//go:build !vks_only

package cmd

// VLB command group + its resource completers. Compiled into the binary by
// default (so dev builds and CI test it), but EXCLUDED from the public release
// binary, which is built with `-tags vks_only` while VLB is still in
// development. Remove this tag (and the build flag in release.yml) once VLB is
// ready to ship.
import (
	_ "github.com/greennodehub/greennode-cli/cmd/vlb"
	_ "github.com/greennodehub/greennode-cli/internal/resources/vlb"
)
