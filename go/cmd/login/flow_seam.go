package login

import (
	"context"
	"path/filepath"

	"github.com/greennodehub/greennode-cli/internal/config"
	loginpkg "github.com/greennodehub/greennode-cli/internal/login"
)

// loginFlow is the slice of *loginpkg.Flow the cmd uses, behind an interface so
// runLogin tests inject a fake flow without binding a real loopback port.
// *loginpkg.Flow satisfies it structurally (AuthURL/RedirectURI/SetAuthorizeURL/
// Complete/Close). Production newFlow returns loginpkg.Prepare's *Flow.
type loginFlow interface {
	AuthURL() string
	RedirectURI() string
	SetAuthorizeURL(url string)
	Complete(ctx context.Context) (loginpkg.Token, error)
	Close() error
}

// newFlow is the cmd→library Prepare seam. Tests override it to return a fake
// loginFlow (no real bind). Mirrors internal/login's openBrowser seam pattern.
// Production delegates to loginpkg.Prepare, whose *Flow satisfies loginFlow.
var newFlow = func(ctx context.Context, cfg loginpkg.Config) (loginFlow, error) {
	return loginpkg.Prepare(ctx, cfg)
}

// openBrowserFn is the cmd→opener seam. Tests no-op it. Production calls
// loginpkg.OpenBrowser (the exported platform opener).
var openBrowserFn = loginpkg.OpenBrowser

// credentialsFilePath returns the per-profile credentials file path under
// ~/.greennode/, used by the cancel test to assert no file was written. It
// mirrors config's own path resolution (DefaultConfigDir is HOME-based and
// honors t.Setenv("HOME", ...) in tests; the profile arg is kept for parity
// with the writer's per-profile layout even though the credentials file is
// shared across profiles in the current INI writer).
func credentialsFilePath(profile string) string {
	return filepath.Join(config.DefaultConfigDir(), "credentials")
}
