package member

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

// memberIntFields / memberBoolFields drive shorthand type coercion for the
// --member struct flag; they mirror MemberRequest.
var (
	memberIntFields  = []string{"port", "weight", "monitorPort"}
	memberBoolFields = []string{"backup"}
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Replace the members of a pool",
	Long: "Set the pool's member list. The API replaces the list wholesale, so any " +
		"existing member not passed via --member is removed. Use --keep-existing to " +
		"add to the current members instead of replacing them.",
	RunE: runUpdate,
}

func init() {
	addIDFlags(updateCmd)
	f := updateCmd.Flags()
	f.StringArray("member", nil,
		"Pool member, shorthand (ipAddress=10.0.0.4,port=80,weight=1,backup=false) or JSON (repeatable)")
	f.Bool("keep-existing", false, "Keep the pool's current members and add the --member entries to them")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the resulting member list without applying it")
}

// currentMemberFields maps a PoolMemberDto field to its MemberRequest name, so
// existing members can be sent back with --keep-existing.
var currentMemberFields = map[string]string{
	"address":      "ipAddress",
	"protocolPort": "port",
	"weight":       "weight",
	"name":         "name",
	"monitorPort":  "monitorPort",
	"backup":       "backup",
}

// buildUpdateBody assembles the ChangeMembersRequest. `current` is the member
// list from GET (nil unless --keep-existing).
func buildUpdateBody(current []interface{}, memberFlags []string, keepExisting bool) (map[string]interface{}, error) {
	members := []map[string]interface{}{}

	if keepExisting {
		for _, item := range current {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			converted := map[string]interface{}{}
			for from, to := range currentMemberFields {
				if v, ok := m[from]; ok && v != nil {
					converted[to] = v
				}
			}
			if len(converted) > 0 {
				members = append(members, converted)
			}
		}
	}

	parsed, err := cli.ParseStructFlags("member", memberFlags, memberIntFields, memberBoolFields)
	if err != nil {
		return nil, err
	}
	// monitorPort is optional in the spec but mandatory in practice: omitting it
	// yields a 400 complaining that monitorPort is out of range, which is
	// baffling when the user never set one. Only new --member entries are
	// checked; members carried over by --keep-existing already have one.
	for i, m := range parsed {
		if _, ok := m["monitorPort"]; !ok {
			return nil, fmt.Errorf("--member[%d]: monitorPort is required by the API "+
				"(set it to the same value as port for a plain health check)", i)
		}
	}
	members = append(members, parsed...)

	if len(members) == 0 && len(memberFlags) == 0 {
		return nil, fmt.Errorf("provide at least one --member, or pass --member '' explicitly to empty the pool")
	}

	return map[string]interface{}{"members": members}, nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	lbID, poolID, err := idsOf(cmd)
	if err != nil {
		return err
	}
	f := cmd.Flags()
	memberFlags, _ := f.GetStringArray("member")
	keepExisting, _ := f.GetBool("keep-existing")
	force, _ := f.GetBool("force")
	dryRun, _ := f.GetBool("dry-run")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	response, err := apiClient.Get(vlbclient.Members(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch the current members of pool %s: %w", poolID, err)
	}
	current := memberList(response)

	body, err := buildUpdateBody(current, memberFlags, keepExisting)
	if err != nil {
		return err
	}

	printMemberPreview(current, body["members"].([]map[string]interface{}))

	if dryRun {
		cli.DryRunNotice("replace the member list")
		return nil
	}
	if !cli.Confirm(force, "Replace the member list of this pool?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Put(vlbclient.Members(projectID, lbID, poolID), body)
	if err != nil {
		return fmt.Errorf("failed to update the members of pool %s: %w", poolID, err)
	}

	return vlbclient.Output(cmd, cfg, result)
}

// memberList extracts the member array from a {"data": [...]} envelope.
func memberList(response interface{}) []interface{} {
	switch v := response.(type) {
	case []interface{}:
		return v
	case map[string]interface{}:
		if d, ok := v["data"].([]interface{}); ok {
			return d
		}
	}
	return nil
}

func printMemberPreview(current []interface{}, resulting []map[string]interface{}) {
	fmt.Printf("The pool's member list will be replaced (%d current -> %d new):\n\n", len(current), len(resulting))
	for _, m := range resulting {
		fmt.Printf("  %v:%v", m["ipAddress"], m["port"])
		if name, ok := m["name"]; ok {
			fmt.Printf("  (%v)", name)
		}
		fmt.Println()
	}
	if len(resulting) == 0 {
		fmt.Println("  (none — the pool will have no members)")
	}
	fmt.Println()
	fmt.Println("Members not listed above are removed from the pool.")
}
