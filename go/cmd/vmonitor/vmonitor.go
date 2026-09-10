// Package vmonitor implements GreenNode vMonitor management commands.
package vmonitor

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

const endpoint = "https://vmonitorapis.vngcloud.vn/vmonitor-api"

var VMonitorCmd = newVMonitorCommand()

func newVMonitorCommand() *cobra.Command {
	root := opengine.NewGroup("vmonitor", "Manage dashboards, alarms, infrastructure monitoring, and metrics")
	root.Long = `Manage GreenNode vMonitor resources through the published global control plane.

The command group covers alarms, dashboards, monitored infrastructure,
integrations, metrics, statistics, views, variables, and widgets.
Use --dry-run to preview changes.`

	groups := map[string]*cobra.Command{
		"alarm":                    opengine.NewGroup("alarm", "Manage alarms"),
		"api-key":                  opengine.NewGroup("api-key", "Manage metric API keys"),
		"change-alarm":             opengine.NewGroup("change-alarm", "Manage change alarms"),
		"dashboard":                opengine.NewGroup("dashboard", "Manage dashboards"),
		"infrastructure":           opengine.NewGroup("infrastructure", "Inspect monitored infrastructure"),
		"integration":              opengine.NewGroup("integration", "Manage monitoring integrations"),
		"metric":                   opengine.NewGroup("metric", "Inspect metric metadata"),
		"metric-unit":              opengine.NewGroup("metric-unit", "List metric units"),
		"metric-unit-mapping":      opengine.NewGroup("metric-unit-mapping", "List metric unit mappings"),
		"metric-unit-mapping-user": opengine.NewGroup("metric-unit-mapping-user", "Manage user metric unit mappings"),
		"statistic":                opengine.NewGroup("statistic", "Query monitoring statistics"),
		"variable":                 opengine.NewGroup("variable", "Manage dashboard variables"),
		"view":                     opengine.NewGroup("view", "Manage dashboard views"),
		"widget":                   opengine.NewGroup("widget", "Manage dashboard widgets"),
		"widget-v2":                opengine.NewGroup("widget-v2", "Manage version-2 dashboard widgets"),
	}
	root.AddCommand(
		groups["alarm"],
		groups["api-key"],
		groups["change-alarm"],
		groups["dashboard"],
		groups["infrastructure"],
		groups["integration"],
		groups["metric"],
		groups["metric-unit"],
		groups["metric-unit-mapping"],
		groups["metric-unit-mapping-user"],
		groups["statistic"],
		groups["variable"],
		groups["view"],
		groups["widget"],
		groups["widget-v2"],
	)
	for _, op := range allOperations() {
		groups[op.Parent].AddCommand(newOperationCommand(op))
	}
	return root
}

func init() {
	cli.RegisterService(VMonitorCmd)
}
