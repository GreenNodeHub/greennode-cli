package member

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Replace the members of a pool",
	Long: "Set the pool's member list. The API replaces the list wholesale, so any " +
		"existing member not passed via --member is removed. Use --keep-existing to " +
		"add to the current members instead of replacing them; a --member with the " +
		"same ipAddress and port as a current member updates only the fields it " +
		"sets on that member (e.g. to change its weight).",
	RunE: runUpdate,
}

func init() {
	addIDFlags(updateCmd)
	f := updateCmd.Flags()
	f.StringArray("member", nil,
		"Pool member, shorthand (ipAddress=10.0.0.4,port=80,monitorPort=80,weight=1,backup=false) or JSON (repeatable)")
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
// memberFlagGiven must come from pflag's Changed, not from len(memberFlags):
// pflag collapses an empty stringArray occurrence, so `--member ”` arrives as an
// empty slice, indistinguishable from the flag never being passed. Guarding on
// the length made the documented way to empty a pool return the very error that
// prescribes it.
func buildUpdateBody(current []interface{}, memberFlags []string, memberFlagGiven, keepExisting bool) (map[string]interface{}, error) {
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

	parsed, err := cli.ParseStructFlags("member", memberFlags, vlbclient.MemberIntFields, vlbclient.MemberBoolFields)
	if err != nil {
		return nil, err
	}
	// --keep-existing with nothing to add would PUT the unchanged list: a no-op
	// that still moves the load balancer into UPDATING and blocks the next
	// mutation. Checked after parsing, so `--keep-existing --member ''` is
	// refused too.
	if keepExisting && len(parsed) == 0 {
		return nil, fmt.Errorf("nothing to update: --keep-existing needs at least one --member")
	}

	// Record the keys the user actually typed before ValidateMembers fills in
	// defaults (backup=false). A merge must overlay only these, or changing a
	// weight would also turn a backup member into a primary.
	explicit := make([][]string, len(parsed))
	for i, m := range parsed {
		for k := range m {
			explicit[i] = append(explicit[i], k)
		}
	}
	// Only new --member entries are checked; members carried over by
	// --keep-existing already carry the service's own values.
	if err := vlbclient.ValidateMembers(parsed); err != nil {
		return nil, err
	}

	// The same ipAddress:port twice in --member is most likely a copy-paste
	// slip (a port meant to differ); refuse it rather than keep one silently.
	seen := map[string]int{}
	for i, m := range parsed {
		key := memberKey(m)
		if j, ok := seen[key]; ok {
			return nil, fmt.Errorf("--member[%d] and --member[%d] are both %s; each member must be given once", j, i, key)
		}
		seen[key] = i
	}

	// A --member with the same ipAddress:port as a carried-over member is merged
	// into it in place: only the keys the user typed change, so
	// `--keep-existing --member ...,weight=5` keeps the member's name and backup
	// flag instead of sending the member twice or wiping them.
	index := map[string]int{}
	for i, m := range members {
		index[memberKey(m)] = i
	}
	for i, m := range parsed {
		if j, ok := index[memberKey(m)]; ok {
			for _, k := range explicit[i] {
				members[j][k] = m[k]
			}
			continue
		}
		members = append(members, m)
	}

	if len(members) == 0 && !memberFlagGiven {
		return nil, fmt.Errorf("provide at least one --member, or pass --member '' explicitly to empty the pool")
	}

	return map[string]interface{}{"members": members}, nil
}

// memberKey identifies a member by ipAddress:port. fmt.Sprint makes a port read
// back from the API (float64 80) and a parsed one (int 80) compare equal.
func memberKey(m map[string]interface{}) string {
	return fmt.Sprint(m["ipAddress"]) + ":" + fmt.Sprint(m["port"])
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

	body, err := buildUpdateBody(current, memberFlags, f.Changed("member"), keepExisting)
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
		fmt.Println("  " + memberPreviewLine(m))
	}
	if len(resulting) == 0 {
		fmt.Println("  (none — the pool will have no members)")
	}
	fmt.Println()
	fmt.Println("Members not listed above are removed from the pool.")
}

// memberPreviewLine renders one member for the confirmation preview. weight and
// backup are shown so that a change to them (for example through a
// --keep-existing merge) is visible before the user confirms.
func memberPreviewLine(m map[string]interface{}) string {
	line := fmt.Sprintf("%v:%v", m["ipAddress"], m["port"])
	if name, ok := m["name"]; ok {
		line += fmt.Sprintf("  (%v)", name)
	}
	if w, ok := m["weight"]; ok {
		line += fmt.Sprintf("  weight=%v", w)
	}
	if b, ok := m["backup"].(bool); ok && b {
		line += "  backup"
	}
	return line
}
