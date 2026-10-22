// Package vmonitorlog implements GreenNode vMonitor Log management commands.
package vmonitorlog

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

const endpoint = "https://vmonitorapis.vngcloud.vn/log-api"

var VMonitorLogCmd = newVMonitorLogCommand()

func newVMonitorLogCommand() *cobra.Command {
	root := opengine.NewGroup("vmonitor-log", "Manage log projects, pipelines, archives, and service mappings")
	root.Long = `Manage GreenNode vMonitor Log resources through the published global API.

The command group covers log projects and search, pipelines and
processors, archive and refill workflows, certificate downloads, and product
log mappings. Use disposable resources when testing writes.`

	groups := map[string]*cobra.Command{
		"archive":                 opengine.NewGroup("archive", "Manage log archives"),
		"certificate":             opengine.NewGroup("certificate", "Manage log project certificates"),
		"log":                     opengine.NewGroup("log", "Search and export project logs"),
		"pipeline":                opengine.NewGroup("pipeline", "Manage log pipelines"),
		"processor":               opengine.NewGroup("processor", "Manage pipeline processors"),
		"processor-group":         opengine.NewGroup("processor-group", "Manage pipeline processor groups"),
		"processor-group-library": opengine.NewGroup("processor-group-library", "Use processor group libraries"),
		"project":                 opengine.NewGroup("project", "Manage log projects"),
		"refill":                  opengine.NewGroup("refill", "Manage log refills"),
		"vcdn-mapping":            opengine.NewGroup("vcdn-mapping", "Manage vCDN log mappings"),
		"vdb-mapping":             opengine.NewGroup("vdb-mapping", "Manage vDB log mappings"),
		"vlb-mapping":             opengine.NewGroup("vlb-mapping", "Manage vLB log mappings"),
		"vstorage-bucket-mapping": opengine.NewGroup("vstorage-bucket-mapping", "Manage vStorage bucket log mappings"),
		"vstorage-mapping":        opengine.NewGroup("vstorage-mapping", "Manage vStorage project log mappings"),
	}
	root.AddCommand(
		groups["archive"],
		groups["certificate"],
		groups["log"],
		groups["pipeline"],
		groups["processor"],
		groups["processor-group"],
		groups["processor-group-library"],
		groups["project"],
		groups["refill"],
		groups["vcdn-mapping"],
		groups["vdb-mapping"],
		groups["vlb-mapping"],
		groups["vstorage-bucket-mapping"],
		groups["vstorage-mapping"],
	)
	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VMonitorLogCmd)
}
