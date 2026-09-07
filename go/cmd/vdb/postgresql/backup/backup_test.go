package backup

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestClusterPathTakesAClusterID pins the shape that is easiest to get wrong in
// this package: the path parameter is the CLUSTER, not the backup. Relational's
// equivalent ("/backups/detail/{backupId}") means the opposite.
func TestClusterPathTakesAClusterID(t *testing.T) {
	cases := map[string]string{
		"/detail":        "/vdb-postgresql/v1/backup/backup-vdb/pg-1/detail",
		"/restore-point": "/vdb-postgresql/v1/backup/backup-vdb/pg-1/restore-point",
		"/backup-now":    "/vdb-postgresql/v1/backup/backup-vdb/pg-1/backup-now",
	}

	for suffix, want := range cases {
		if got := clusterPath("pg-1", suffix); got != want {
			t.Errorf("clusterPath(%q) = %q, want %q", suffix, got, want)
		}
	}
}

func TestRequireClusterIDRejectsInstanceIDs(t *testing.T) {
	if err := requireClusterID("pg-2e6f2253-9032-466f-975f-d6d6b6ec8330"); err != nil {
		t.Errorf("valid cluster ID rejected: %v", err)
	}
	for _, id := range []string{"db-66a37ca3-e688-4a8a-9e35-a89fa69733c6", "bk-db-1", ""} {
		if err := requireClusterID(id); err == nil {
			t.Errorf("requireClusterID(%q) = nil, want an error", id)
		}
	}
}

func TestCommandsAreWired(t *testing.T) {
	want := map[string]bool{"list": true, "get": true, "list-restore-points": true, "create": true}
	for _, sub := range BackupCmd.Commands() {
		delete(want, sub.Name())
	}
	for name := range want {
		t.Errorf("backup %s is not registered", name)
	}

	// Taking a backup consumes billable backup storage, so it is gated like the
	// other mutating vdb commands.
	for _, flag := range []string{"dry-run", "force"} {
		if createCmd.Flags().Lookup(flag) == nil {
			t.Errorf("backup create must define --%s", flag)
		}
	}
}

func TestFlagCompletionsAreRegistered(t *testing.T) {
	for _, cmd := range []*cobra.Command{getCmd, listRestorePointsCmd, createCmd} {
		if cmd.Flags().Lookup("cluster-id") == nil {
			t.Errorf("%s has no --cluster-id flag", cmd.Name())
			continue
		}
		if _, ok := cmd.GetFlagCompletionFunc("cluster-id"); !ok {
			t.Errorf("%s --cluster-id has no completion function registered", cmd.Name())
		}
	}
}
