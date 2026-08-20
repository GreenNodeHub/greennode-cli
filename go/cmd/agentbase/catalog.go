package agentbase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	catalogpkg "github.com/greennodehub/greennode-cli/internal/agentbase/catalog"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
)

// catalogCmd groups the marketplace commands (was `catalog`; renamed to match
// the portal). The marketplace surface (compute flavors, openclaw versions, and
// openclaw CRUD/lifecycle) shares the runtime client's base URL.
var catalogCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse the marketplace (flavors, openclaw versions, openclaws)",
	Long: `Browse and manage the marketplace catalog.

Two groups:

  flavors            — compute flavors (cpu/ram/supportedResourceTypes), the
                        runtime-side catalog. (Distinct from 'gateway flavors',
                        which are the gateway placement flavors.)
  openclaw-versions  — available OpenClaw runtime versions.
  openclaw           — OpenClaw instances: list/get/create/delete + start/stop
                        and switch version.

Marketplace ops share the ~/.greennode profile and the runtime endpoint.`,
}

// catalogFlavorsCmd groups the flavor commands.
var catalogFlavorsCmd = &cobra.Command{
	Use:   "flavors",
	Short: "Browse compute flavors",
}

// catalogOpenClawVersionsCmd groups the openclaw-version commands.
var catalogOpenClawVersionsCmd = &cobra.Command{
	Use:   "openclaw-versions",
	Short: "Browse OpenClaw versions",
}

// catalogOpenClawCmd groups the openclaw lifecycle commands.
var catalogOpenClawCmd = &cobra.Command{
	Use:   "openclaw",
	Short: "Manage OpenClaws",
}

// newCatalogClient mirrors newRuntimeClient (catalog is served by the runtime
// service, so it uses ab.endpoints.Runtime).
func newCatalogClient(ctx context.Context, cmd *cobra.Command) (*catalogpkg.Client, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return catalogpkg.NewClient(ab.endpoints.Runtime, provider), nil
}

// ---------------------------------------------------------------------------
// flavors list
// ---------------------------------------------------------------------------

var catalogFlavorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List compute flavors",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resourceType, _ := cmd.Flags().GetString("resource-type")
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		flavors, err := client.ListFlavors(ctx, resourceType)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(flavors)
		case output.FormatID:
			if len(flavors) > 0 {
				output.PrintID(flavors[0].ID)
			}
			return nil
		}
		if len(flavors) == 0 {
			fmt.Fprintln(os.Stderr, "No flavors found.")
			return nil
		}
		rows := make([][]string, 0, len(flavors))
		for i := range flavors {
			f := flavors[i]
			rows = append(rows, []string{
				f.ID, f.Name,
				fmt.Sprintf("%d", f.CPU), fmt.Sprintf("%d", f.RAM),
				strings.Join(f.SupportedResourceTypes, ","),
				fmt.Sprintf("%t", f.Enabled),
			})
		}
		output.Table([]string{"ID", "Name", "CPU", "RAM", "Resource Types", "Enabled"}, rows)
		return nil
	},
}

// ---------------------------------------------------------------------------
// openclaw-versions list
// ---------------------------------------------------------------------------

var catalogOpenClawVersionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OpenClaw versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		versions, err := client.ListOpenClawVersions(ctx)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(versions)
		case output.FormatID:
			if len(versions) > 0 {
				output.PrintID(versions[0].ID)
			}
			return nil
		}
		if len(versions) == 0 {
			fmt.Fprintln(os.Stderr, "No openclaw versions found.")
			return nil
		}
		rows := make([][]string, 0, len(versions))
		for i := range versions {
			v := versions[i]
			rows = append(rows, []string{v.ID, v.Name, fmt.Sprintf("%t", v.DefaultVersion)})
		}
		output.Table([]string{"ID", "Name", "Default"}, rows)
		return nil
	},
}

// ---------------------------------------------------------------------------
// openclaw list
// ---------------------------------------------------------------------------

var catalogOpenClawListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OpenClaws",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListOpenClaws(ctx, page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.ListData) > 0 {
				output.PrintID(resp.ListData[0].ID)
			}
			return nil
		}
		if len(resp.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No openclaws found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.ListData))
		for i := range resp.ListData {
			o := resp.ListData[i]
			rows = append(rows, []string{
				o.ID, o.Name, o.VersionID, o.FlavorID, o.Status,
				fmt.Sprintf("%t", o.POC), formatTimeVal(o.CreatedAt),
			})
		}
		output.Table([]string{"ID", "Name", "Version", "Flavor", "Status", "POC", "Created"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

// ---------------------------------------------------------------------------
// openclaw create
// ---------------------------------------------------------------------------

// catalogFile is shared by create (--file); only one command runs at a time.
var catalogFile string

var catalogOpenClawCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new OpenClaw",
	Long: `Create a new OpenClaw.

The minimal flag path (name + version-id + flavor-id) creates a bare openclaw.
For anything richer — messaging channels (telegram/zalo), the GreenNode model
provider, environment variables, or POC flag — write a spec file and apply it
with --file (keys are the JSON/camelCase field names). Example:

    name: my-bot
    versionId: v1
    flavorId: f1
    poc: true
    greenNodeModelProvider:
      enabled: true
      apiKeyName: my-key
    environmentVariables:
      LOG_LEVEL: info
    channels:
      telegram:
        botToken: tok
        dmPolicy: ALLOWLIST
        dmAllowedUserIds: ["123"]`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if catalogFile != "" {
			data, err := os.ReadFile(catalogFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			req, err := loadOpenClawSpec(data)
			if err != nil {
				return err
			}
			return createOpenClawAndPrint(ctx, cmd, req)
		}
		f := cmd.Flags()
		name, _ := f.GetString("name")
		versionID, _ := f.GetString("version-id")
		flavorID, _ := f.GetString("flavor-id")
		if name == "" || versionID == "" || flavorID == "" {
			return fmt.Errorf("required flags not set: --name, --version-id, --flavor-id (or use --file)")
		}
		req := &catalogpkg.OpenClawCreateRequest{Name: name, VersionID: versionID, FlavorID: flavorID}
		if f.Changed("poc") {
			poc, _ := f.GetBool("poc")
			req.POC = poc
		}
		return createOpenClawAndPrint(ctx, cmd, req)
	},
}

func createOpenClawAndPrint(ctx context.Context, cmd *cobra.Command, req *catalogpkg.OpenClawCreateRequest) error {
	client, err := newCatalogClient(ctx, cmd)
	if err != nil {
		return err
	}
	oc, err := client.CreateOpenClaw(ctx, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "OpenClaw %q created (id %s, status %s).\n", oc.Name, oc.ID, oc.Status)
	return output.PrintResource(oc, func() string { return oc.ID }, func() error { return renderOpenClawDetail(oc) })
}

// ---------------------------------------------------------------------------
// openclaw get
// ---------------------------------------------------------------------------

var catalogOpenClawGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show an OpenClaw",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		oc, err := client.GetOpenClaw(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(oc, func() string { return oc.ID }, func() error { return renderOpenClawDetail(oc) })
	},
}

// ---------------------------------------------------------------------------
// openclaw delete
// ---------------------------------------------------------------------------

var catalogOpenClawDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an OpenClaw",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteOpenClaw(ctx, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "OpenClaw %q deleted.\n", args[0])
		output.PrintDeletedID(args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// openclaw start / stop
// ---------------------------------------------------------------------------

var catalogOpenClawStartCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start an OpenClaw",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.StartOpenClaw(ctx, args[0]); err != nil {
			return err
		}
		output.Successf("OpenClaw %s started.", args[0])
		return nil
	},
}

var catalogOpenClawStopCmd = &cobra.Command{
	Use:   "stop <id>",
	Short: "Stop an OpenClaw",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.StopOpenClaw(ctx, args[0]); err != nil {
			return err
		}
		output.Successf("OpenClaw %s stopped.", args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// openclaw update-version
// ---------------------------------------------------------------------------

var catalogOpenClawUpdateVersionCmd = &cobra.Command{
	Use:   "update-version <id>",
	Short: "Switch an OpenClaw to a different version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		versionID, _ := cmd.Flags().GetString("version-id")
		if versionID == "" {
			return fmt.Errorf("required flag %q not set", "version-id")
		}
		client, err := newCatalogClient(ctx, cmd)
		if err != nil {
			return err
		}
		oc, err := client.UpdateOpenClawVersion(ctx, args[0], versionID)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "OpenClaw %s switched to version %s.\n", args[0], oc.VersionID)
		return output.PrintResource(oc, func() string { return oc.ID }, func() error { return renderOpenClawDetail(oc) })
	},
}

// ---------------------------------------------------------------------------
// rendering helpers
// ---------------------------------------------------------------------------

func renderOpenClawDetail(oc *catalogpkg.OpenClawDto) error {
	rows := [][]string{
		{"ID", oc.ID},
		{"Name", oc.Name},
		{"Version ID", oc.VersionID},
		{"URL", output.StrOrDash(oc.URL)},
		{"Gateway Token", output.StrOrDash(oc.GatewayToken)},
		{"GreenNode API Key", output.StrOrDash(oc.GreenNodeApiKeyName)},
		{"Flavor ID", oc.FlavorID},
		{"Status", oc.Status},
		{"POC", fmt.Sprintf("%t", oc.POC)},
		{"Created", formatTimeVal(oc.CreatedAt)},
		{"Updated", formatTimeVal(oc.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// ---------------------------------------------------------------------------
// file parsing (YAML or JSON -> struct)
// ---------------------------------------------------------------------------

// loadOpenClawSpec parses a YAML/JSON create spec into OpenClawCreateRequest.
// yaml→map→json→struct so the file's camelCase keys bind to the struct's json
// tags (yaml.v3 does not honor json tags directly).
func loadOpenClawSpec(data []byte) (*catalogpkg.OpenClawCreateRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req catalogpkg.OpenClawCreateRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid openclaw spec: %w", err)
	}
	if req.Name == "" || req.VersionID == "" || req.FlavorID == "" {
		return nil, fmt.Errorf("spec is missing required field(s): name, versionId, flavorId")
	}
	return &req, nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	AgentbaseCmd.AddCommand(catalogCmd)

	// flavors group
	catalogFlavorsListCmd.Flags().String("resource-type", "", "Filter by supported resource type")
	catalogFlavorsCmd.AddCommand(catalogFlavorsListCmd)
	catalogCmd.AddCommand(catalogFlavorsCmd)

	// openclaw-versions group
	catalogOpenClawVersionsCmd.AddCommand(catalogOpenClawVersionsListCmd)
	catalogCmd.AddCommand(catalogOpenClawVersionsCmd)

	// openclaw group
	catalogOpenClawListCmd.Flags().Int("page", 1, "Page number (1-based)")
	catalogOpenClawListCmd.Flags().Int("size", 10, "Page size")
	catalogOpenClawCmd.AddCommand(catalogOpenClawListCmd)

	catalogOpenClawCreateCmd.Flags().String("name", "", "OpenClaw name (required without --file)")
	catalogOpenClawCreateCmd.Flags().String("version-id", "", "OpenClaw version id (required without --file)")
	catalogOpenClawCreateCmd.Flags().String("flavor-id", "", "Flavor id (required without --file)")
	catalogOpenClawCreateCmd.Flags().Bool("poc", false, "Mark as proof-of-concept")
	catalogOpenClawCreateCmd.Flags().StringVar(&catalogFile, "file", "", "Apply a spec file (authoritative when set)")
	catalogOpenClawCmd.AddCommand(catalogOpenClawCreateCmd)

	catalogOpenClawCmd.AddCommand(catalogOpenClawGetCmd)
	catalogOpenClawCmd.AddCommand(catalogOpenClawDeleteCmd)
	catalogOpenClawCmd.AddCommand(catalogOpenClawStartCmd)
	catalogOpenClawCmd.AddCommand(catalogOpenClawStopCmd)

	catalogOpenClawUpdateVersionCmd.Flags().String("version-id", "", "Target version id (required)")
	catalogOpenClawCmd.AddCommand(catalogOpenClawUpdateVersionCmd)

	catalogCmd.AddCommand(catalogOpenClawCmd)
}
