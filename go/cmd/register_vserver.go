//go:build !vks_only

package cmd

// vServer command group + its resource completers. Compiled into the binary by
// default (so dev builds and CI test it), but EXCLUDED from the public release
// binary, which is built with `-tags vks_only` while vServer is still in
// development. Remove this tag (and the build flag in release.yml) once vServer
// is ready to ship.
import (
	_ "github.com/greennodehub/greennode-cli/cmd/vserver"
	_ "github.com/greennodehub/greennode-cli/internal/resources/vserver"
)
