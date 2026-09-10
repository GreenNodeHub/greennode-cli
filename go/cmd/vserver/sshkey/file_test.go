package sshkey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateKeyFileNeverOverwritesOrFollowsLinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fixture-target")
	if err := os.WriteFile(target, []byte("fixture-original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "fixture-key.pem")); err != nil {
		t.Fatal(err)
	}
	path, err := savePrivateKey("fixture-key", "fixture-private", dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "fixture-key (1).pem" {
		t.Fatalf("path = %s", path)
	}
	if original, err := os.ReadFile(target); err != nil || string(original) != "fixture-original" {
		t.Fatalf("target changed: %q, %v", original, err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "fixture-private\n" {
		t.Fatalf("key = %q, %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".grn-key-") {
			t.Fatal("temporary key remains")
		}
	}
}

func TestPrivateKeyFileRejectsUnsafeNamesBeforeCreatingDirectories(t *testing.T) {
	for _, name := range []string{"", " ", ".", "..", "../fixture", "fixture/key", `fixture\key`, "fixture\x00key"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "fixture-output")
			if _, err := savePrivateKey(name, "fixture-private", dir); err == nil {
				t.Fatal("unsafe name accepted")
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatalf("output directory created: %v", err)
			}
		})
	}
}
