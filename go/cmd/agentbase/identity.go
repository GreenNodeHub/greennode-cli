package agentbase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/auth"
	"github.com/vngcloud/greennode-cli/internal/agentbase/cliinput"
	"github.com/vngcloud/greennode-cli/internal/agentbase/config"
	identitypkg "github.com/vngcloud/greennode-cli/internal/agentbase/identity"
	"github.com/vngcloud/greennode-cli/internal/agentbase/jsonslice"
	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
)

// --- root identity command ---

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage authentication and agent identities",
	Long:  `Login, logout, manage agent identities, and configure outbound authentication providers.`,
}

// --- identity login ---

var identityLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the GreenNode platform",
	Long: `Authenticate with the GreenNode AgentBase platform using OAuth2 client credentials.

Your Client ID and Client Secret are saved to ./.greennode.json.

Credentials can be supplied via flags, environment variables (GREENNODE_CLIENT_ID /
GREENNODE_CLIENT_SECRET), or interactively with --interactive.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		clientID, _ := cmd.Flags().GetString("client-id")
		if clientID == "" {
			clientID = os.Getenv("GREENNODE_CLIENT_ID")
		}
		secret, _ := cmd.Flags().GetString("client-secret")
		if secret == "" {
			secret = os.Getenv("GREENNODE_CLIENT_SECRET")
		}

		var err error
		clientID, err = cliinput.RequireOrPromptString(clientID, "--client-id", "Client ID")
		if err != nil {
			return err
		}
		secret, err = cliinput.RequireOrPromptSecret(secret, "--client-secret", "Client Secret")
		if err != nil {
			return err
		}

		// Validate credentials by fetching a token.
		cfg, _ := config.Load()
		tokenURL := cfg.Endpoints.OAuth2Token
		provider := auth.NewProvider(clientID, secret, tokenURL)
		if _, err := provider.AccessToken(context.Background()); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		if err := config.SaveCredentials(clientID, secret); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		output.Success("Logged in successfully")
		return nil
	},
}

// --- identity logout ---

var identityLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	Long: `Remove your Client ID and Client Secret from ./.greennode.json.

If you set them via GREENNODE_CLIENT_ID / GREENNODE_CLIENT_SECRET environment variables,
you must unset those separately.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ClearCredentials(); err != nil {
			return err
		}
		output.Success("Logged out. Credentials cleared from ./.greennode.json")
		return nil
	},
}

// --- identity whoami ---

var identityWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently active credentials",
	Long:  `Display the current environment, client ID, and agent identity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustLoadConfig()
		output.Table([]string{"Field", "Value"}, [][]string{
			{"Environment", string(cfg.Env)},
			{"Client ID", output.StrOrDash(cfg.ClientID)},
			{"Agent Identity", output.StrOrDash(cfg.AgentIdentity)},
		})
		return nil
	},
}

// --- identity config ---

var identityConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show identity configuration",
	Long:  `Display the current identity configuration including endpoints and credentials.`,
}

var identityConfigShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current configuration",
	Long:  `Display the current identity configuration including environment, credentials, and endpoint URLs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustLoadConfig()
		secret := "-"
		if cfg.ClientSecret != "" {
			secret = "***" + cfg.ClientSecret[max(0, len(cfg.ClientSecret)-4):]
		}
		output.Table([]string{"Key", "Value"}, [][]string{
			{"environment", string(cfg.Env)},
			{"client_id", output.StrOrDash(cfg.ClientID)},
			{"client_secret", secret},
			{"agent_identity", output.StrOrDash(cfg.AgentIdentity)},
			{"identity_url", cfg.Endpoints.Identity},
			{"runtime_url", cfg.Endpoints.Runtime},
			{"memory_url", cfg.Endpoints.Memory},
			{"oauth2_token_url", cfg.Endpoints.OAuth2Token},
		})
		return nil
	},
}

// --- identity workload ---

var workloadCmd = &cobra.Command{
	Use:   "workload",
	Short: "Manage agent workload identities",
	Long:  `Create, list, get, update, and delete agent workload identities used to represent digital identities for agents accessing external services.`,
}

var (
	workloadCreateName       string
	workloadCreateSetCurrent bool
)

var workloadCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent identity",
	Long: `Create a new agent identity for the authenticated user.

Agent identities are used to represent digital identities for agents accessing
external services. The name must be 3-50 characters and match the pattern
^[a-zA-Z0-9_-]+$.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		workloadCreateName, err = cliinput.RequireOrPromptString(workloadCreateName, "--name", "Agent identity name")
		if err != nil {
			return err
		}

		desc, _ := cmd.Flags().GetString("description")
		urls, _ := cmd.Flags().GetStringArray("allowed-return-url")

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		req := &identitypkg.CreateAgentIdentityRequest{Name: workloadCreateName}
		if cmd.Flags().Changed("description") {
			req.Description = &desc
		}
		if cmd.Flags().Changed("allowed-return-url") {
			req.AllowedReturnURLs = jsonslice.Array[string](urls)
		}
		identity, err := client.CreateAgentIdentity(ctx, req)
		if err != nil {
			return err
		}
		if workloadCreateSetCurrent {
			if err := config.SaveAgentIdentity(str(identity.Name)); err != nil {
				output.Warn("Identity created but failed to save as current: " + err.Error())
			}
		}
		return output.PrintResource(identity, func() string { return str(identity.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(identity.ID))},
				{"Name", output.StrOrDash(str(identity.Name))},
				{"Description", output.StrOrDash(str(identity.Description))},
				{"Created", formatTime(identity.CreatedAt)},
				{"Updated", formatTime(identity.UpdatedAt)},
			})
			fmt.Fprintln(os.Stdout, "\nAllowed Return URLs:")
			if len(identity.AllowedReturnURLs) == 0 {
				fmt.Fprintln(os.Stdout, "  (none)")
			} else {
				rows := make([][]string, len(identity.AllowedReturnURLs))
				for i, u := range identity.AllowedReturnURLs {
					rows[i] = []string{u}
				}
				output.Table([]string{"URL"}, rows)
			}
			return nil
		})
	},
}

var workloadListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agent identities",
	Long:  `Retrieve a paginated list of all agent identities owned by the authenticated user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.ListAgentIdentities(ctx, page-1, size)
		if err != nil {
			return err
		}

		switch output.GetFormat() {
		case output.FormatTable:
			if len(resp.Content) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No items found.")
				return nil
			}
			rows := make([][]string, len(resp.Content))
			for i, id := range resp.Content {
				rows[i] = []string{str(id.ID), str(id.Name), output.StrOrDash(str(id.Description))}
			}
			output.Table([]string{"ID", "Name", "Description"}, rows)
			if resp.Page != nil && resp.TotalPages != nil && resp.TotalElements != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Page %d of %d (%d total items)\n", *resp.Page+1, *resp.TotalPages, *resp.TotalElements)
			}
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.Content) > 0 {
				output.PrintID(str(resp.Content[0].ID))
			}
		}
		return nil
	},
}

var workloadGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get an agent identity by name",
	Long:  `Retrieve a specific agent identity by its name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		id, err := client.GetAgentIdentity(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(id, func() string { return str(id.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(id.ID))},
				{"Name", output.StrOrDash(str(id.Name))},
				{"Description", output.StrOrDash(str(id.Description))},
				{"Created", formatTime(id.CreatedAt)},
				{"Updated", formatTime(id.UpdatedAt)},
			})
			fmt.Fprintln(os.Stdout, "\nAllowed Return URLs:")
			if len(id.AllowedReturnURLs) == 0 {
				fmt.Fprintln(os.Stdout, "  (none)")
			} else {
				rows := make([][]string, len(id.AllowedReturnURLs))
				for i, u := range id.AllowedReturnURLs {
					rows[i] = []string{u}
				}
				output.Table([]string{"URL"}, rows)
			}
			return nil
		})
	},
}

var workloadUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update an agent identity",
	Long:  `Update an existing agent identity. Only description and allowed return URLs can be modified.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		desc, _ := cmd.Flags().GetString("description")
		urls, _ := cmd.Flags().GetStringArray("allowed-return-url")
		req := &identitypkg.UpdateAgentIdentityRequest{}
		if cmd.Flags().Changed("description") {
			req.Description = &desc
		}
		if cmd.Flags().Changed("allowed-return-url") {
			req.AllowedReturnURLs = jsonslice.Array[string](urls)
		}
		id, err := client.UpdateAgentIdentity(ctx, args[0], req)
		if err != nil {
			return err
		}
		return output.PrintResource(id, func() string { return str(id.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(id.ID))},
				{"Name", output.StrOrDash(str(id.Name))},
				{"Description", output.StrOrDash(str(id.Description))},
				{"Created", formatTime(id.CreatedAt)},
				{"Updated", formatTime(id.UpdatedAt)},
			})
			fmt.Fprintln(os.Stdout, "\nAllowed Return URLs:")
			if len(id.AllowedReturnURLs) == 0 {
				fmt.Fprintln(os.Stdout, "  (none)")
			} else {
				rows := make([][]string, len(id.AllowedReturnURLs))
				for i, u := range id.AllowedReturnURLs {
					rows[i] = []string{u}
				}
				output.Table([]string{"URL"}, rows)
			}
			return nil
		})
	},
}

var workloadUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the current agent identity",
	Long:  `Set the given agent identity name as the current identity in ./.greennode.json.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.SaveAgentIdentity(args[0]); err != nil {
			return err
		}
		output.Successf("Current agent identity set to: %s", args[0])
		return nil
	},
}

var workloadDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an agent identity",
	Long:  `Delete an agent identity by name. The identity will be soft-deleted.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		if err := client.DeleteAgentIdentity(ctx, args[0]); err != nil {
			return err
		}
		return output.PrintDeletedID(args[0])
	},
}

// --- outbound-auth ---

var outboundAuthCmd = &cobra.Command{
	Use:   "outbound-auth",
	Short: "Manage outbound authentication providers",
	Long:  `Manage static API key, delegated API key, and OAuth2 providers for agent outbound authentication.`,
}

// --- outbound-auth static ---

var staticCmd = &cobra.Command{
	Use:   "static",
	Short: "Manage static API key providers",
	Long:  `Create, list, get, update, and delete static API key providers for outbound authentication.`,
}

var staticCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a static API key provider",
	Long: `Create a new static API key provider. The name must be 3-50 characters and match
the pattern ^[a-zA-Z0-9_-]+$. Both name and API key are required.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		var err error
		name, err = cliinput.RequireOrPromptString(name, "--name", "Provider name")
		if err != nil {
			return err
		}

		apikey, _ := cmd.Flags().GetString("apikey")
		apikey, err = cliinput.RequireOrPromptSecret(apikey, "--apikey", "API Key")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.CreateApikeyProvider(ctx, &identitypkg.CreateApikeyProviderRequest{Name: name, Apikey: apikey})
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var staticListCmd = &cobra.Command{
	Use:   "list",
	Short: "List static API key providers",
	Long:  `Retrieve a paginated list of static API key providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.ListApikeyProviders(ctx, page-1, size)
		if err != nil {
			return err
		}

		switch output.GetFormat() {
		case output.FormatTable:
			if len(resp.Content) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No items found.")
				return nil
			}
			rows := make([][]string, len(resp.Content))
			for i, p := range resp.Content {
				rows[i] = []string{str(p.ID), str(p.Name), str(p.Status)}
			}
			output.Table([]string{"ID", "Name", "Status"}, rows)
			if resp.Page != nil && resp.TotalPages != nil && resp.TotalElements != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Page %d of %d (%d total items)\n", *resp.Page+1, *resp.TotalPages, *resp.TotalElements)
			}
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.Content) > 0 {
				output.PrintID(str(resp.Content[0].ID))
			}
		}
		return nil
	},
}

var staticGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a static API key provider",
	Long:  `Retrieve a static API key provider by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.GetApikeyProvider(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var staticUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a static API key provider",
	Long:  `Update the API key value of an existing static API key provider.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apikey, _ := cmd.Flags().GetString("apikey")
		var err error
		apikey, err = cliinput.RequireOrPromptSecret(apikey, "--apikey", "New API Key")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		if err := client.UpdateApikeyProvider(ctx, args[0], &identitypkg.UpdateApikeyProviderRequest{Apikey: apikey}); err != nil {
			return err
		}
		// Fetch updated resource for detail output
		resp, err := client.GetApikeyProvider(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var staticDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a static API key provider",
	Long:  `Delete a static API key provider by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		if err := client.DeleteApikeyProvider(ctx, args[0]); err != nil {
			return err
		}
		return output.PrintDeletedID(args[0])
	},
}

var staticGetKeyCmd = &cobra.Command{
	Use:   "get-key <provider-name> <identity-name>",
	Short: "Get the API key for an agent identity",
	Long:  `Retrieve the API key assigned to a specific agent identity from a static API key provider.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.GetApikeyForAgentIdentity(ctx, args[0], args[1])
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatTable:
			output.Table([]string{"Field", "Value"}, [][]string{
				{"API Key", output.StrOrDash(str(resp.Apikey))},
			})
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			output.PrintID(str(resp.Apikey))
		}
		return nil
	},
}

// --- outbound-auth delegated ---

var delegatedCmd = &cobra.Command{
	Use:   "delegated",
	Short: "Manage delegated API key providers",
	Long:  `Create, list, get, and delete delegated API key providers for user-provided API keys for external services.`,
}

var delegatedCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a delegated API key provider",
	Long: `Create a new delegated API key provider. The name must be 3-50 characters and
match the pattern ^[a-zA-Z0-9_-]+$.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		var err error
		name, err = cliinput.RequireOrPromptString(name, "--name", "Provider name")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.CreateDelegatedProvider(ctx, &identitypkg.CreateDelegatedApiKeyProviderRequest{Name: name})
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var delegatedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List delegated API key providers",
	Long:  `Retrieve a paginated list of delegated API key providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.ListDelegatedProviders(ctx, page-1, size)
		if err != nil {
			return err
		}

		switch output.GetFormat() {
		case output.FormatTable:
			if len(resp.Content) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No items found.")
				return nil
			}
			rows := make([][]string, len(resp.Content))
			for i, p := range resp.Content {
				rows[i] = []string{str(p.ID), str(p.Name), str(p.Status)}
			}
			output.Table([]string{"ID", "Name", "Status"}, rows)
			if resp.Page != nil && resp.TotalPages != nil && resp.TotalElements != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Page %d of %d (%d total items)\n", *resp.Page+1, *resp.TotalPages, *resp.TotalElements)
			}
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.Content) > 0 {
				output.PrintID(str(resp.Content[0].ID))
			}
		}
		return nil
	},
}

var delegatedGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a delegated API key provider",
	Long:  `Retrieve a delegated API key provider by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.GetDelegatedProvider(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var delegatedDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a delegated API key provider",
	Long:  `Delete a delegated API key provider by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		if err := client.DeleteDelegatedProvider(ctx, args[0]); err != nil {
			return err
		}
		return output.PrintDeletedID(args[0])
	},
}

var delegatedGetKeyCmd = &cobra.Command{
	Use:   "get-key <provider-name> <identity-name>",
	Short: "Obtain a delegated API key for an agent identity",
	Long: `Obtain a delegated API key for an agent identity from a delegated API key provider.

Required flags: --agent-user-id and --return-url.
Optional flags: --custom-state, --session-id, --force-delegation.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentUserID, _ := cmd.Flags().GetString("agent-user-id")
		returnURL, _ := cmd.Flags().GetString("return-url")

		var err error
		agentUserID, err = cliinput.RequireOrPromptString(agentUserID, "--agent-user-id", "Agent user ID")
		if err != nil {
			return err
		}
		returnURL, err = cliinput.RequireOrPromptString(returnURL, "--return-url", "Return URL")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		req := &identitypkg.GetDelegatedApiKeyRequest{
			AgentUserID: agentUserID,
			ReturnURL:   returnURL,
		}
		if cmd.Flags().Changed("custom-state") {
			v, _ := cmd.Flags().GetString("custom-state")
			req.CustomState = &v
		}
		if cmd.Flags().Changed("session-id") {
			v, _ := cmd.Flags().GetString("session-id")
			req.SessionID = &v
		}
		if cmd.Flags().Changed("force-delegation") {
			v, _ := cmd.Flags().GetBool("force-delegation")
			req.ForceDelegation = &v
		}
		resp, err := client.GetDelegatedApiKey(ctx, args[0], args[1], req)
		if err != nil {
			return err
		}

		switch output.GetFormat() {
		case output.FormatTable:
			output.Table([]string{"Field", "Value"}, [][]string{
				{"API Key", output.StrOrDash(str(resp.Apikey))},
				{"Authorization URL", output.StrOrDash(str(resp.AuthorizationURL))},
				{"Session ID", output.StrOrDash(str(resp.SessionID))},
				{"Status", output.StrOrDash(str(resp.Status))},
			})
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			output.PrintID(str(resp.SessionID))
		}
		return nil
	},
}

// --- outbound-auth oauth2 ---

var oauth2Cmd = &cobra.Command{
	Use:   "oauth2",
	Short: "Manage OAuth2 providers",
	Long:  `Create, list, get, update, and delete OAuth2 providers, and retrieve M2M and 3-legged OAuth2 tokens.`,
}

var oauth2CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an OAuth2 provider",
	Long: `Create a new OAuth2 provider. The name must be 3-50 characters and match the
pattern ^[a-zA-Z0-9_-]+$. All of name, client-id, client-secret, authorization-url,
and token-url are required.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		clientID, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")
		authURL, _ := cmd.Flags().GetString("authorization-url")
		tokenURL, _ := cmd.Flags().GetString("token-url")

		var err error
		name, err = cliinput.RequireOrPromptString(name, "--name", "Provider name")
		if err != nil {
			return err
		}
		clientID, err = cliinput.RequireOrPromptString(clientID, "--client-id", "Client ID")
		if err != nil {
			return err
		}
		clientSecret, err = cliinput.RequireOrPromptSecret(clientSecret, "--client-secret", "Client Secret")
		if err != nil {
			return err
		}
		authURL, err = cliinput.RequireOrPromptString(authURL, "--authorization-url", "Authorization URL")
		if err != nil {
			return err
		}
		tokenURL, err = cliinput.RequireOrPromptString(tokenURL, "--token-url", "Token URL")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		req := &identitypkg.CreateOauth2ProviderRequest{
			Name:             name,
			ClientID:         clientID,
			ClientSecret:     clientSecret,
			AuthorizationURL: authURL,
			TokenURL:         tokenURL,
		}
		resp, err := client.CreateOauth2Provider(ctx, req)
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Client ID", output.StrOrDash(str(resp.ClientID))},
				{"Authorization URL", output.StrOrDash(str(resp.AuthorizationURL))},
				{"Token URL", output.StrOrDash(str(resp.TokenURL))},
				{"Callback URL", output.StrOrDash(str(resp.CallbackURL))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var oauth2ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OAuth2 providers",
	Long:  `Retrieve a paginated list of OAuth2 providers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.ListOauth2Providers(ctx, page-1, size)
		if err != nil {
			return err
		}

		switch output.GetFormat() {
		case output.FormatTable:
			if len(resp.Content) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No items found.")
				return nil
			}
			rows := make([][]string, len(resp.Content))
			for i, p := range resp.Content {
				rows[i] = []string{str(p.ID), str(p.Name), str(p.Status), str(p.AuthorizationURL)}
			}
			output.Table([]string{"ID", "Name", "Status", "Auth URL"}, rows)
			if resp.Page != nil && resp.TotalPages != nil && resp.TotalElements != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Page %d of %d (%d total items)\n", *resp.Page+1, *resp.TotalPages, *resp.TotalElements)
			}
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.Content) > 0 {
				output.PrintID(str(resp.Content[0].ID))
			}
		}
		return nil
	},
}

var oauth2GetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get an OAuth2 provider",
	Long:  `Retrieve an OAuth2 provider by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.GetOauth2Provider(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Client ID", output.StrOrDash(str(resp.ClientID))},
				{"Authorization URL", output.StrOrDash(str(resp.AuthorizationURL))},
				{"Token URL", output.StrOrDash(str(resp.TokenURL))},
				{"Callback URL", output.StrOrDash(str(resp.CallbackURL))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var oauth2UpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update an OAuth2 provider",
	Long:  `Update an existing OAuth2 provider. All of client-id, client-secret, authorization-url, and token-url are required.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clientID, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")
		authURL, _ := cmd.Flags().GetString("authorization-url")
		tokenURL, _ := cmd.Flags().GetString("token-url")

		var err error
		clientID, err = cliinput.RequireOrPromptString(clientID, "--client-id", "Client ID")
		if err != nil {
			return err
		}
		clientSecret, err = cliinput.RequireOrPromptSecret(clientSecret, "--client-secret", "Client Secret")
		if err != nil {
			return err
		}
		authURL, err = cliinput.RequireOrPromptString(authURL, "--authorization-url", "Authorization URL")
		if err != nil {
			return err
		}
		tokenURL, err = cliinput.RequireOrPromptString(tokenURL, "--token-url", "Token URL")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		req := &identitypkg.UpdateOauth2ProviderRequest{
			ClientID:         clientID,
			ClientSecret:     clientSecret,
			AuthorizationURL: authURL,
			TokenURL:         tokenURL,
		}
		if err := client.UpdateOauth2Provider(ctx, args[0], req); err != nil {
			return err
		}
		// Fetch updated resource for detail output
		resp, err := client.GetOauth2Provider(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(resp, func() string { return str(resp.Name) }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"ID", output.StrOrDash(str(resp.ID))},
				{"Name", output.StrOrDash(str(resp.Name))},
				{"Status", output.StrOrDash(str(resp.Status))},
				{"Client ID", output.StrOrDash(str(resp.ClientID))},
				{"Authorization URL", output.StrOrDash(str(resp.AuthorizationURL))},
				{"Token URL", output.StrOrDash(str(resp.TokenURL))},
				{"Callback URL", output.StrOrDash(str(resp.CallbackURL))},
				{"Created", formatTime(resp.CreatedAt)},
				{"Updated", formatTime(resp.UpdatedAt)},
			})
			return nil
		})
	},
}

var oauth2DeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an OAuth2 provider",
	Long:  `Delete an OAuth2 provider by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		if err := client.DeleteOauth2Provider(ctx, args[0]); err != nil {
			return err
		}
		return output.PrintDeletedID(args[0])
	},
}

var oauth2M2MTokenCmd = &cobra.Command{
	Use:   "m2m-token <provider-name> <identity-name>",
	Short: "Get an M2M OAuth2 token",
	Long:  `Retrieve a machine-to-machine (client credentials) OAuth2 token for an agent identity via an OAuth2 provider.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		scopes, _ := cmd.Flags().GetStringArray("scope")
		var err error
		scopes, err = cliinput.RequireOrPromptStringSlice(scopes, "--scope", "OAuth2 scopes")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		resp, err := client.GetM2MToken(ctx, args[0], args[1], &identitypkg.GetM2mTokenRequest{Scopes: jsonslice.Array[string](scopes)})
		if err != nil {
			return err
		}

		switch output.GetFormat() {
		case output.FormatTable:
			output.Table([]string{"Field", "Value"}, [][]string{
				{"Access Token", output.StrOrDash(str(resp.AccessToken))},
				{"Token Type", output.StrOrDash(str(resp.TokenType))},
			})
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			output.PrintID(str(resp.AccessToken))
		}
		return nil
	},
}

var oauth23LOTokenCmd = &cobra.Command{
	Use:   "3lo-token <provider-name> <identity-name>",
	Short: "Get a 3-legged OAuth2 token",
	Long: `Retrieve a 3-legged OAuth2 token for an agent identity via an OAuth2 provider.

Required flags: --agent-user-id, --return-url, --scope.
Optional flags: --session-id, --custom-parameters, --custom-state, --force-authentication.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentUserID, _ := cmd.Flags().GetString("agent-user-id")
		returnURL, _ := cmd.Flags().GetString("return-url")
		scopes, _ := cmd.Flags().GetStringArray("scope")

		var err error
		agentUserID, err = cliinput.RequireOrPromptString(agentUserID, "--agent-user-id", "Agent user ID")
		if err != nil {
			return err
		}
		returnURL, err = cliinput.RequireOrPromptString(returnURL, "--return-url", "Return URL")
		if err != nil {
			return err
		}
		scopes, err = cliinput.RequireOrPromptStringSlice(scopes, "--scope", "OAuth2 scopes")
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := newIdentityClient(ctx)
		if err != nil {
			return err
		}
		req := &identitypkg.ThreeLoTokenRequest{
			AgentUserID: agentUserID,
			ReturnURL:   returnURL,
			Scopes:      jsonslice.Array[string](scopes),
		}
		if cmd.Flags().Changed("session-id") {
			v, _ := cmd.Flags().GetString("session-id")
			req.SessionID = &v
		}
		if cmd.Flags().Changed("custom-state") {
			v, _ := cmd.Flags().GetString("custom-state")
			req.CustomState = &v
		}
		if cmd.Flags().Changed("force-authentication") {
			v, _ := cmd.Flags().GetBool("force-authentication")
			req.ForceAuthentication = &v
		}
		customParams, _ := cmd.Flags().GetString("custom-parameters")
		if customParams != "" {
			var params map[string]string
			if err := json.Unmarshal([]byte(customParams), &params); err != nil {
				return fmt.Errorf("invalid --custom-parameters JSON: %w", err)
			}
			req.CustomParameters = &params
		}

		resp, err := client.Get3LOToken(ctx, args[0], args[1], req)
		if err != nil {
			return err
		}

		switch output.GetFormat() {
		case output.FormatTable:
			output.Table([]string{"Field", "Value"}, [][]string{
				{"Access Token", output.StrOrDash(str(resp.AccessToken))},
				{"Token Type", output.StrOrDash(str(resp.TokenType))},
				{"Authorization URL", output.StrOrDash(str(resp.AuthorizationURL))},
				{"Session ID", output.StrOrDash(str(resp.SessionID))},
				{"Status", output.StrOrDash(str(resp.Status))},
			})
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			output.PrintID(str(resp.AccessToken))
		}
		return nil
	},
}

func init() {
	AgentbaseCmd.AddCommand(identityCmd)

	identityLoginCmd.Flags().String("client-id", "", "OAuth2 client ID (env: GREENNODE_CLIENT_ID)")
	identityLoginCmd.Flags().String("client-secret", "", "OAuth2 client secret (env: GREENNODE_CLIENT_SECRET)")
	identityCmd.AddCommand(identityLoginCmd)
	identityCmd.AddCommand(identityLogoutCmd)
	identityCmd.AddCommand(identityWhoamiCmd)

	identityCmd.AddCommand(identityConfigCmd)
	identityConfigCmd.AddCommand(identityConfigShowCmd)

	// workload
	identityCmd.AddCommand(workloadCmd)

	workloadCreateCmd.Flags().StringVarP(&workloadCreateName, "name", "n", "", "Agent identity name (required without --interactive)")
	workloadCreateCmd.Flags().BoolVar(&workloadCreateSetCurrent, "set-current", false, "Set as the current agent identity after creation")
	workloadCreateCmd.Flags().String("description", "", "Description of the agent identity")
	workloadCreateCmd.Flags().StringArray("allowed-return-url", nil, "Allowed return URL (repeatable)")
	workloadCmd.AddCommand(workloadCreateCmd)

	workloadListCmd.Flags().Int("page", 1, "Page number (1-based)")
	workloadListCmd.Flags().Int("size", 20, "Page size")
	workloadCmd.AddCommand(workloadListCmd)

	workloadCmd.AddCommand(workloadGetCmd)

	workloadUpdateCmd.Flags().String("description", "", "Updated description")
	workloadUpdateCmd.Flags().StringArray("allowed-return-url", nil, "Allowed return URL (repeatable)")
	workloadCmd.AddCommand(workloadUpdateCmd)

	workloadCmd.AddCommand(workloadUseCmd)
	workloadCmd.AddCommand(workloadDeleteCmd)

	// outbound-auth
	identityCmd.AddCommand(outboundAuthCmd)

	// static
	outboundAuthCmd.AddCommand(staticCmd)

	staticCreateCmd.Flags().StringP("name", "n", "", "Provider name (required without --interactive)")
	staticCreateCmd.Flags().String("apikey", "", "API key value (required without --interactive)")
	staticCmd.AddCommand(staticCreateCmd)

	staticListCmd.Flags().Int("page", 1, "Page number (1-based)")
	staticListCmd.Flags().Int("size", 20, "Page size")
	staticCmd.AddCommand(staticListCmd)

	staticCmd.AddCommand(staticGetCmd)

	staticUpdateCmd.Flags().String("apikey", "", "New API key value (required without --interactive)")
	staticCmd.AddCommand(staticUpdateCmd)

	staticCmd.AddCommand(staticDeleteCmd)
	staticCmd.AddCommand(staticGetKeyCmd)

	// delegated
	outboundAuthCmd.AddCommand(delegatedCmd)

	delegatedCreateCmd.Flags().StringP("name", "n", "", "Provider name (required without --interactive)")
	delegatedCmd.AddCommand(delegatedCreateCmd)

	delegatedListCmd.Flags().Int("page", 1, "Page number (1-based)")
	delegatedListCmd.Flags().Int("size", 20, "Page size")
	delegatedCmd.AddCommand(delegatedListCmd)

	delegatedCmd.AddCommand(delegatedGetCmd)
	delegatedCmd.AddCommand(delegatedDeleteCmd)

	delegatedGetKeyCmd.Flags().String("agent-user-id", "", "Agent user ID (required without --interactive)")
	delegatedGetKeyCmd.Flags().String("return-url", "", "Return URL after authorization (required without --interactive)")
	delegatedGetKeyCmd.Flags().String("custom-state", "", "Custom state parameter")
	delegatedGetKeyCmd.Flags().String("session-id", "", "Session ID (UUID format)")
	delegatedGetKeyCmd.Flags().Bool("force-delegation", false, "Force delegation")
	delegatedCmd.AddCommand(delegatedGetKeyCmd)

	// oauth2
	outboundAuthCmd.AddCommand(oauth2Cmd)

	oauth2CreateCmd.Flags().StringP("name", "n", "", "Provider name (required without --interactive)")
	oauth2CreateCmd.Flags().String("client-id", "", "OAuth2 client ID (required without --interactive)")
	oauth2CreateCmd.Flags().String("client-secret", "", "OAuth2 client secret (required without --interactive)")
	oauth2CreateCmd.Flags().String("authorization-url", "", "Authorization endpoint URL (required without --interactive)")
	oauth2CreateCmd.Flags().String("token-url", "", "Token endpoint URL (required without --interactive)")
	oauth2Cmd.AddCommand(oauth2CreateCmd)

	oauth2ListCmd.Flags().Int("page", 1, "Page number (1-based)")
	oauth2ListCmd.Flags().Int("size", 20, "Page size")
	oauth2Cmd.AddCommand(oauth2ListCmd)

	oauth2Cmd.AddCommand(oauth2GetCmd)

	oauth2UpdateCmd.Flags().String("client-id", "", "OAuth2 client ID (required without --interactive)")
	oauth2UpdateCmd.Flags().String("client-secret", "", "OAuth2 client secret (required without --interactive)")
	oauth2UpdateCmd.Flags().String("authorization-url", "", "Authorization endpoint URL (required without --interactive)")
	oauth2UpdateCmd.Flags().String("token-url", "", "Token endpoint URL (required without --interactive)")
	oauth2Cmd.AddCommand(oauth2UpdateCmd)

	oauth2Cmd.AddCommand(oauth2DeleteCmd)

	oauth2M2MTokenCmd.Flags().StringArray("scope", nil, "OAuth2 scope (repeatable, required without --interactive)")
	oauth2Cmd.AddCommand(oauth2M2MTokenCmd)

	oauth23LOTokenCmd.Flags().String("agent-user-id", "", "Agent user ID (required without --interactive)")
	oauth23LOTokenCmd.Flags().String("return-url", "", "Return URL after authorization (required without --interactive)")
	oauth23LOTokenCmd.Flags().StringArray("scope", nil, "OAuth2 scope (repeatable, required without --interactive)")
	oauth23LOTokenCmd.Flags().String("session-id", "", "Session ID (UUID format)")
	oauth23LOTokenCmd.Flags().String("custom-parameters", "", `Custom parameters as a JSON object, e.g. '{"key1":"value1"}'`)
	oauth23LOTokenCmd.Flags().String("custom-state", "", "Custom state parameter")
	oauth23LOTokenCmd.Flags().Bool("force-authentication", false, "Force re-authentication")
	oauth2Cmd.AddCommand(oauth23LOTokenCmd)
}

// str safely dereferences a string pointer, returning an empty string if nil.
func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// joinStrings joins a JSON slice of strings with the given separator.
func joinStrings(urls jsonslice.Array[string], sep string) string {
	return strings.Join([]string(urls), sep)
}

// formatTime formats a time pointer for display, returning "-" if nil.
func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// newIdentityClient builds an authenticated identity client.
func newIdentityClient(ctx context.Context) (*identitypkg.Client, error) {
	cfg := mustLoadConfigWithCreds()
	provider := newAuthProvider(cfg)
	if _, err := provider.AccessToken(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return identitypkg.NewClient(cfg.Endpoints.Identity, provider), nil
}
