package login

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoad_RoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "token.json")
	in := StoredToken{RefreshToken: "rt-xyz", ExpiresAt: time.UnixMilli(1700000000000).UTC()}
	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.RefreshToken != "rt-xyz" {
		t.Errorf("RefreshToken=%q, want rt-xyz", out.RefreshToken)
	}
	if !out.ExpiresAt.Equal(in.ExpiresAt) {
		t.Errorf("ExpiresAt=%v, want %v", out.ExpiresAt, in.ExpiresAt)
	}
}

func TestSave_FilePerms0600_Dir0700(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested", "token.json")
	if err := Save(sub, StoredToken{RefreshToken: "rt"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	fi, err := os.Stat(sub)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Errorf("file perm=%o, want 0600", perm)
	}
	di, err := os.Stat(filepath.Join(dir, "nested"))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := di.Mode().Perm(); perm != 0700 {
		t.Errorf("dir perm=%o, want 0700", perm)
	}
}

func TestLoad_MissingFileIsError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nope.json")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error loading missing file, got nil")
	}
	if !errors.Is(err, ErrNoStoredToken) {
		t.Errorf("Load missing file: want ErrNoStoredToken, got %v", err)
	}
}

func TestClear_RemovesFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "token.json")
	if err := Save(path, StoredToken{RefreshToken: "rt"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Clear(path); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file gone after Clear; stat err=%v", err)
	}
}

func TestClear_MissingFileIsNoError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "never.json")
	if err := Clear(path); err != nil {
		t.Errorf("Clear on missing file should be nil; got %v", err)
	}
}
