package agentbase

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/cliinput"
	crpkg "github.com/vngcloud/greennode-cli/internal/agentbase/cr"
	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
)

// crCmd groups the agent-core-container-registry commands. The service wraps
// VNG Cloud Container Registry (vCR): each user gets an auto-provisioned
// repository + robot account on first access, so there is no create — resources
// appear when first read. The agentbase /cr endpoint fronts
// agent-core-container-registry's /api/v1.
var crCmd = &cobra.Command{
	Use:   "cr",
	Short: "Manage the container registry (repository, images, artifacts, robot credential)",
	Long: `Manage the agentbase container registry (agent-core-container-registry).

A user's repository and robot account are auto-provisioned on first read, so
there is no 'create' — they appear when first fetched. You can list/delete
images and the artifacts (digests) inside them, and fetch or rotate the robot
account used for 'docker login':

    grn agentbase cr repository get
    grn agentbase cr repository image list
    grn agentbase cr repository artifact list --image-name myapp
    grn agentbase cr registry-credential get
    grn agentbase cr registry-credential reset-secret

The robot-account secret is real (it authorizes push/pull): it is MASKED in
table output and revealed only with -o json. Use 'reset-secret' to rotate it.`,
}

// newCRClient mirrors newPolicyClient: resolve the shared profile + env, select
// the shared token provider, force-mint once so auth failures surface before the
// first call, and point the typed client at the cr endpoint.
func newCRClient(ctx context.Context, cmd *cobra.Command) (*crpkg.Client, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return crpkg.NewClient(ab.endpoints.Cr, provider), nil
}

// ---------------------------------------------------------------------------
// repository
// ---------------------------------------------------------------------------

var crRepositoryCmd = &cobra.Command{
	Use:   "repository",
	Short: "Show the user's container-registry repository",
	Long: `Show the user's auto-provisioned repository (name, registry URL, image count,
quota). The repository + robot account are created on first access if absent.`,
}

var crRepositoryGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the user's repository info",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCRClient(ctx, cmd)
		if err != nil {
			return err
		}
		repo, err := client.GetRepository(ctx)
		if err != nil {
			return err
		}
		return output.PrintResource(repo, func() string { return repo.Name }, func() error { return renderRepository(repo) })
	},
}

// ---------------------------------------------------------------------------
// repository image
// ---------------------------------------------------------------------------

var crImageCmd = &cobra.Command{
	Use:   "image",
	Short: "List or delete images in the user's repository",
}

var crImageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List images",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		page, _ := f.GetInt("page")
		size, _ := f.GetInt("size")
		name, _ := f.GetString("name")
		client, err := newCRClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListImages(ctx, name, page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			for _, img := range resp.Data {
				output.PrintID(img.Name)
			}
			return nil
		}
		if len(resp.Data) == 0 {
			fmt.Fprintln(os.Stderr, "No images found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Data))
		for i := range resp.Data {
			img := resp.Data[i]
			rows = append(rows, []string{img.Name, fmt.Sprintf("%d", img.ArtifactCount), fmt.Sprintf("%d", img.PullCount), formatTimeVal(img.UpdateTime)})
		}
		output.Table([]string{"Name", "Artifacts", "Pulls", "Updated"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

var crImageDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an image (all its artifacts/tags)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		imageName, _ := f.GetString("image-name")
		imageName, err := cliinput.RequireOrPromptString(imageName, "--image-name", "Image name to delete")
		if err != nil {
			return err
		}
		client, err := newCRClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteImage(ctx, imageName); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Image %q deleted.\n", imageName)
		return output.PrintDeletedID(imageName)
	},
}

// ---------------------------------------------------------------------------
// repository artifact
// ---------------------------------------------------------------------------

var crArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "List or delete artifacts (digests) within an image",
}

var crArtifactListCmd = &cobra.Command{
	Use:   "list",
	Short: "List artifacts within an image",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		imageName, _ := f.GetString("image-name")
		imageName, err := cliinput.RequireOrPromptString(imageName, "--image-name", "Image name (required)")
		if err != nil {
			return err
		}
		page, _ := f.GetInt("page")
		size, _ := f.GetInt("size")
		name, _ := f.GetString("name")
		client, err := newCRClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListArtifacts(ctx, imageName, name, page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			for _, a := range resp.Data {
				output.PrintID(a.Digest)
			}
			return nil
		}
		if len(resp.Data) == 0 {
			fmt.Fprintln(os.Stderr, "No artifacts found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Data))
		for i := range resp.Data {
			a := resp.Data[i]
			rows = append(rows, []string{a.Digest, a.Type, formatBytes(a.Size), tagsToStr(a.Tags), formatTimeVal(a.PushTime)})
		}
		output.Table([]string{"Digest", "Type", "Size", "Tags", "Pushed"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

var crArtifactDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a single artifact by digest",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		imageName, _ := f.GetString("image-name")
		imageName, err := cliinput.RequireOrPromptString(imageName, "--image-name", "Image name (required)")
		if err != nil {
			return err
		}
		digest, _ := f.GetString("digest")
		digest, err = cliinput.RequireOrPromptString(digest, "--digest", "Artifact digest (required)")
		if err != nil {
			return err
		}
		client, err := newCRClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteArtifact(ctx, imageName, digest); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Artifact %s deleted (image %q).\n", digest, imageName)
		return output.PrintDeletedID(digest)
	},
}

// ---------------------------------------------------------------------------
// registry-credential (robot account)
// ---------------------------------------------------------------------------

var crRegistryCredentialCmd = &cobra.Command{
	Use:   "registry-credential",
	Short: "Manage the robot account used for docker login",
	Long: `Manage the robot account (username + secret) used to authenticate to the
registry for push/pull. The secret is MASKED in table output and revealed only
with -o json (so you can pipe it into 'docker login'). 'reset-secret' rotates
the secret.`,
}

var crRegistryCredentialGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the robot account (username + secret)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCRClient(ctx, cmd)
		if err != nil {
			return err
		}
		cred, err := client.GetRegistryCredential(ctx)
		if err != nil {
			return err
		}
		return output.PrintResource(cred, func() string { return cred.Username }, func() error { return renderRegistryCredential(cred) })
	},
}

var crRegistryCredentialResetSecretCmd = &cobra.Command{
	Use:   "reset-secret",
	Short: "Rotate the robot-account secret",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newCRClient(ctx, cmd)
		if err != nil {
			return err
		}
		cred, err := client.ResetSecret(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Secret rotated. The previous secret is now invalid.")
		return output.PrintResource(cred, func() string { return cred.Username }, func() error { return renderRegistryCredential(cred) })
	},
}

// ---------------------------------------------------------------------------
// rendering helpers
// ---------------------------------------------------------------------------

func renderRepository(repo *crpkg.Repository) error {
	rows := [][]string{
		{"Name", repo.Name},
		{"Registry URL", output.StrOrDash(repo.RegistryURL)},
		{"Image Count", fmt.Sprintf("%d", repo.ImageCount)},
		{"Quota Used", fmt.Sprintf("%d", repo.QuotaUsed)},
		{"Quota Limit", fmt.Sprintf("%d", repo.QuotaLimit)},
		{"Created", formatTimeVal(repo.CreatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// renderRegistryCredential masks the secret in the human table; the JSON path
// (output.FormatJSON) emits the raw struct and reveals it.
func renderRegistryCredential(cred *crpkg.RegistryCredential) error {
	rows := [][]string{
		{"Username", cred.Username},
		{"Secret", maskSecret(cred.Secret)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	fmt.Fprintln(os.Stderr, "Secret is masked. Use -o json to reveal it for 'docker login'.")
	return nil
}

// maskSecret shows only the last 4 chars of a secret. Mirrors the
// refresh_token masking posture used elsewhere in the CLI.
func maskSecret(s string) string {
	if s == "" {
		return "-"
	}
	if len(s) <= 4 {
		return "****"
	}
	return "********" + s[len(s)-4:]
}

// tagsToStr joins artifact tag names (comma-separated) for table display.
func tagsToStr(tags []crpkg.Tag) string {
	if len(tags) == 0 {
		return "-"
	}
	out := ""
	for i, t := range tags {
		if i > 0 {
			out += ","
		}
		out += t.Name
	}
	return out
}

// formatBytes renders a byte count in human units for table display.
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	AgentbaseCmd.AddCommand(crCmd)

	// repository
	crCmd.AddCommand(crRepositoryCmd)
	crRepositoryCmd.AddCommand(crRepositoryGetCmd)

	// repository image
	crRepositoryCmd.AddCommand(crImageCmd)
	crImageListCmd.Flags().Int("page", 1, "Page number (1-based)")
	crImageListCmd.Flags().Int("size", 10, "Page size")
	crImageListCmd.Flags().String("name", "", "Filter by image name (case-insensitive substring)")
	crImageCmd.AddCommand(crImageListCmd)

	crImageDeleteCmd.Flags().String("image-name", "", "Image name to delete (required)")
	crImageCmd.AddCommand(crImageDeleteCmd)

	// repository artifact
	crRepositoryCmd.AddCommand(crArtifactCmd)
	crArtifactListCmd.Flags().String("image-name", "", "Image name (required)")
	crArtifactListCmd.Flags().Int("page", 1, "Page number (1-based)")
	crArtifactListCmd.Flags().Int("size", 10, "Page size")
	crArtifactListCmd.Flags().String("name", "", "Filter by digest/tag (case-insensitive substring)")
	crArtifactCmd.AddCommand(crArtifactListCmd)

	crArtifactDeleteCmd.Flags().String("image-name", "", "Image name (required)")
	crArtifactDeleteCmd.Flags().String("digest", "", "Artifact digest (required)")
	crArtifactCmd.AddCommand(crArtifactDeleteCmd)

	// registry-credential
	crCmd.AddCommand(crRegistryCredentialCmd)
	crRegistryCredentialCmd.AddCommand(crRegistryCredentialGetCmd)
	crRegistryCredentialCmd.AddCommand(crRegistryCredentialResetSecretCmd)
}
