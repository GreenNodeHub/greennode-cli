package vks

import (
	"reflect"
	"sort"
	"testing"

	"github.com/spf13/pflag"
)

func TestExtendedVKSCommandsRegistered(t *testing.T) {
	want := map[string][]string{
		"acknowledge-kubeconfig-warning":  {"cluster-id", "dry-run"},
		"create-workspace":                {"dry-run"},
		"get-nodegroup-events":            {"action", "cluster-id", "nodegroup-id", "page", "page-size", "type"},
		"get-upgrade-insights":            {"cluster-id", "page", "page-size"},
		"get-workspace":                   {},
		"list-nodegroup-images":           {},
		"register-fleet":                  {"cluster-id", "dry-run", "enable-east-west-traffic", "enable-north-south-traffic", "fleet-id", "fleet-name", "fleet-type"},
		"reset-workspace-service-account": {"dry-run", "force"},
		"stop-poc":                        {"cluster-id", "dry-run", "force"},
		"unregister-fleet":                {"cluster-id", "dry-run", "force"},
	}
	for path, expected := range want {
		command, args, err := VksCmd.Find([]string{path})
		if err != nil || len(args) != 0 || command.Name() != path {
			t.Fatalf("%s is not registered: %v", path, err)
		}
		got := []string{}
		command.LocalNonPersistentFlags().VisitAll(func(flag *pflag.Flag) { got = append(got, flag.Name) })
		sort.Strings(got)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s flags = %v, want %v", path, got, expected)
		}
	}
}
