package login

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// StoredToken is the persisted subset of a login result: only what must survive
// across CLI invocations. The access token is intentionally absent — it is
// never written to disk.
type StoredToken struct {
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// ErrNoStoredToken is returned by Load when no token file exists.
var ErrNoStoredToken = errors.New("login: no stored token")

// Save writes t as JSON to path with 0600 perms, creating the parent dir with
// 0700. The write is atomic (temp file + rename in the same dir): a crash
// mid-write cannot truncate an existing token, and the rename re-asserts 0600
// even when the target already exists with looser perms (os.WriteFile preserves
// the existing file's mode bits on overwrite). Mirrors greennode-cli
// internal/agentbase/config/config.go:231 (0600 file) and
// internal/config/writer.go:23 (0700 dir).
func Save(path string, t StoredToken) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".token-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	remove := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		remove()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		remove()
		return err
	}
	if err := tmp.Close(); err != nil {
		remove()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		remove()
		return err
	}
	return nil
}

// Load reads and decodes the token file. A missing file returns ErrNoStoredToken
// (not a generic os error) so callers can distinguish "not logged in" from a
// disk failure.
func Load(path string) (StoredToken, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return StoredToken{}, ErrNoStoredToken
	}
	if err != nil {
		return StoredToken{}, err
	}
	var t StoredToken
	if err := json.Unmarshal(b, &t); err != nil {
		return StoredToken{}, err
	}
	return t, nil
}

// Clear removes the token file. A missing file is not an error (idempotent).
func Clear(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
