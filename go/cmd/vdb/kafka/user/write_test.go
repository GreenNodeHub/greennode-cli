package user

import (
	"os"
	"path/filepath"
	"testing"
)

// An unset --*-topics flag must reach the API as [] and not as null: the API
// rejects null for these array fields — measured live, creating a user without
// topic permissions answers 400 bad_request.
func TestTopicNamesNeverNil(t *testing.T) {
	if got := topicNames(""); got == nil || len(got) != 0 {
		t.Errorf("topicNames(\"\") = %#v, want an empty non-nil slice", got)
	}
	got := topicNames("orders, payments")
	if len(got) != 2 || got[0] != "orders" || got[1] != "payments" {
		t.Errorf("topicNames(\"orders, payments\") = %#v", got)
	}
}

// The credential ZIP lands at --output-file when given and at the user-id default
// otherwise, with 0600 permissions: the archive carries private keys and passwords.
func TestWriteCredsZip(t *testing.T) {
	dir := t.TempDir()

	named := filepath.Join(dir, "creds.zip")
	got, err := writeCredsZip(named, "user-1", "PK\x03\x04rest-of-archive")
	if err != nil || got != named {
		t.Fatalf("writeCredsZip(named) = %q, %v", got, err)
	}
	info, err := os.Stat(named)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("permissions of %s = %v (err %v), want 0600", named, info.Mode().Perm(), err)
	}

	// Empty --output-file falls back to kafka-creds-<user-id>.zip in the cwd, so
	// chdir into the temp dir for the default-name case.
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Skipf("cannot chdir: %v", err)
	}
	defer os.Chdir(wd) //nolint:errcheck

	got, err = writeCredsZip("", "user-1", "PK\x03\x04")
	if err != nil {
		t.Fatalf("writeCredsZip(default) error: %v", err)
	}
	if got != "kafka-creds-user-1.zip" {
		t.Errorf("writeCredsZip(default) = %q, want kafka-creds-user-1.zip", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Errorf("default file not written: %v", err)
	}
}
