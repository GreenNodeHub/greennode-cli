//go:build !vks_only

package cmd

// vDB command group. Compiled into the binary by default (so dev builds and CI
// test it), but EXCLUDED from the public release binary, which is built with
// `-tags vks_only` while vDB is still in development. Remove this tag (and the
// build flag in release.yml) once vDB is ready to ship.
import (
	_ "github.com/greennodehub/greennode-cli/cmd/vdb"
)
