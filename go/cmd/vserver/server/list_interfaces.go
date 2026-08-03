package server

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/formatter"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var listInterfacesCmd = &cobra.Command{
	Use:   "list-interfaces",
	Short: "List the network interfaces attached to a server",
	Long: `List the network interfaces attached to a vServer instance.

Internal (PRIVATE) and external (PUBLIC) interfaces are shown separately. In
table output they are rendered as two tables; other formats return the full
response unchanged.`,
	RunE: runListInterfaces,
}

func init() {
	listInterfacesCmd.Flags().String("server-id", "", "Server ID (required)")
	listInterfacesCmd.MarkFlagRequired("server-id") //nolint:errcheck
}

// uuidPreviewLen is how many runes of an id are shown in table output.
const uuidPreviewLen = 20

// interfaceColumns is the column order shown for each interface table. Fields not
// listed are hidden from the table but remain in JSON output.
var interfaceColumns = []string{"uuid", "fixedIp", "floatingIp", "status", "interfaceType", "subnetUuid", "mac", "createdAt"}

func runListInterfaces(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}
	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/servers/%s/network-interfaces", projectID, serverID), nil)
	if err != nil {
		return fmt.Errorf("failed to list network interfaces for server %s: %w", serverID, err)
	}

	return outputServerInterfaces(cmd, cfg, result)
}

// outputServerInterfaces prints the server's interfaces. In table mode it renders two
// labelled tables (internal then external); other formats show the full response.
func outputServerInterfaces(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	if resolveOutput(cmd, cfg) != "table" {
		return outputResult(cmd, cfg, result)
	}

	data := interfacesData(result)
	internal := transformInterfaceRows(interfaceArray(data, "internalInterfaces"))
	external := transformInterfaceRows(interfaceArray(data, "externalInterfaces"))

	printInterfaceTable("Internal interfaces", internal)
	fmt.Fprintln(os.Stdout)
	printInterfaceTable("External interfaces", external)
	return nil
}

// printInterfaceTable prints a labelled count header followed by the interface table,
// or "(none)" when the list is empty.
func printInterfaceTable(title string, rows []interface{}) {
	fmt.Fprintf(os.Stdout, "%s (%d):\n\n", title, len(rows))
	if len(rows) == 0 {
		fmt.Fprintln(os.Stdout, "(none)")
		return
	}
	formatter.FormatTableWithColumns(rows, interfaceColumns, "", os.Stdout) //nolint:errcheck
}

// interfacesData unwraps the {"data": {...}} envelope, returning the inner object that
// holds the internalInterfaces / externalInterfaces arrays.
func interfacesData(result interface{}) map[string]interface{} {
	if v, ok := result.(map[string]interface{}); ok {
		if d, ok := v["data"].(map[string]interface{}); ok {
			return d
		}
		return v
	}
	return nil
}

// interfaceArray returns the named array field from the interfaces object.
func interfaceArray(data map[string]interface{}, key string) []interface{} {
	if data == nil {
		return nil
	}
	if arr, ok := data[key].([]interface{}); ok {
		return arr
	}
	return nil
}

// transformInterfaceRows shortens the uuid and subnetUuid and formats timestamps for
// table display, leaving the underlying JSON untouched.
func transformInterfaceRows(items []interface{}) []interface{} {
	rows := make([]interface{}, 0, len(items))
	for _, it := range items {
		obj, ok := it.(map[string]interface{})
		if !ok {
			rows = append(rows, it)
			continue
		}
		out := make(map[string]interface{}, len(obj))
		for k, val := range obj {
			switch {
			case k == "uuid" || k == "subnetUuid":
				if s, ok := val.(string); ok {
					out[k] = formatter.Truncate(s, uuidPreviewLen)
					continue
				}
				out[k] = val
			case k == "createdAt" || k == "updatedAt":
				if s, ok := val.(string); ok {
					out[k] = formatter.ShortDate(s)
					continue
				}
				out[k] = val
			default:
				out[k] = val
			}
		}
		rows = append(rows, out)
	}
	return rows
}
