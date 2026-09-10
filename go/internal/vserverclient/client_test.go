package vserverclient

import (
	"testing"

	"github.com/greennodehub/greennode-cli/internal/config"
)

func TestProjectIDRejectsPathInjection(t *testing.T) {
	for _, id := range []string{"", "../fixture-project", "fixture/project", "fixture?project"} {
		if _, err := ProjectID(&config.Config{ProjectID: id}); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	if id, err := ProjectID(&config.Config{ProjectID: "fixture-project"}); err != nil || id != "fixture-project" {
		t.Fatalf("project = %s: %v", id, err)
	}
}
