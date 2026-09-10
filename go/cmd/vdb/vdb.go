package vdb

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

var VDBCmd = newVDBCommand()

func newVDBCommand() *cobra.Command {
	root := opengine.NewGroup("vdb", "Manage relational, MemoryStore, Kafka, and PostgreSQL databases")
	root.Long = `Manage GreenNode vDB resources through the published global gateway.

The command group covers relational databases, MemoryStore databases,
Kafka clusters, and PostgreSQL clusters. Use disposable resources when testing
writes because database, storage, backup, and cluster operations can incur cost.`

	families := map[string]*cobra.Command{
		"relational": opengine.NewGroup("relational", "Manage relational database resources"),
		"memory":     opengine.NewGroup("memory", "Manage MemoryStore resources"),
		"kafka":      opengine.NewGroup("kafka", "Manage Kafka resources"),
		"postgresql": opengine.NewGroup("postgresql", "Manage PostgreSQL cluster resources"),
	}
	root.AddCommand(families["relational"], families["memory"], families["kafka"], families["postgresql"])

	groups := make(map[string]*cobra.Command)
	add := func(family, name, short string) {
		group := opengine.NewGroup(name, short)
		families[family].AddCommand(group)
		groups[family+"-"+name] = group
	}
	for _, family := range []string{"relational", "memory"} {
		add(family, "instance", "Manage database instances")
		add(family, "backup", "Manage database backups")
		add(family, "backup-storage", "Manage backup storage")
		add(family, "configuration", "Manage configuration groups")
		add(family, "catalog", "Inspect database plans and network metadata")
	}
	add("kafka", "cluster", "Manage Kafka clusters")
	add("kafka", "topic", "Manage Kafka topics")
	add("kafka", "user", "Manage Kafka users")
	add("kafka", "configuration", "Manage Kafka configuration groups")
	add("kafka", "catalog", "Inspect Kafka plans and metadata")
	add("postgresql", "cluster", "Manage PostgreSQL clusters")
	add("postgresql", "backup", "Manage PostgreSQL backups")

	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VDBCmd)
}
