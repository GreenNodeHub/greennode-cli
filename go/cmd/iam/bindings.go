package iam

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newIAMBindingCommand(action, short string, buildPath readPath, selfTargetFlag string) *cobra.Command {
	return &cobra.Command{
		Use:   action,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := buildPath(cmd)
			if err != nil {
				return err
			}
			if err := rejectIAMMutationEndpointOverride(cmd); err != nil {
				return err
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if dryRun {
				cli.PrintDryRun(action, "IAM access binding", map[string]any{
					"method": mutationMethod(action),
					"path":   path,
				})
				return nil
			}

			force, _ := cmd.Flags().GetBool("force")
			if !cli.Confirm(force, fmt.Sprintf("%s IAM access binding at %s? This changes effective permissions.", title(action), path)) {
				fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return cli.ConfirmationError()
			}

			if selfTargetFlag != "" {
				if err := rejectCurrentIAMIdentityTarget(cmd, action, selfTargetFlag); err != nil {
					return err
				}
			}

			apiClient, err := createPoliciesClient(cmd)
			if err != nil {
				return err
			}

			response, err := apiClient.RequestWithStatusNoRetrySensitive(mutationMethod(action), path, nil, nil)
			if err != nil {
				return fmt.Errorf("failed to %s IAM access binding: %w", action, err)
			}
			if response.StatusCode != 204 || !response.Empty {
				return fmt.Errorf("IAM binding requires an empty HTTP 204 response")
			}
			return outputResult(cmd, cli.RedactJSON(response.Data))
		},
	}
}

func addIAMBindingFlags(cmd *cobra.Command, ids ...struct {
	name  string
	usage string
}) {
	for _, id := range ids {
		addRequiredIDFlag(cmd, id.name, id.usage)
	}
	cmd.Flags().Bool("dry-run", false, "Preview the IAM access change without executing it")
	cmd.Flags().Bool("force", false, "Skip the IAM access-change confirmation prompt")
}

func rejectIAMMutationEndpointOverride(cmd *cobra.Command) error {
	endpointURL, _ := cmd.Flags().GetString("endpoint-url")
	if endpointURL != "" {
		return fmt.Errorf("--endpoint-url is not supported for IAM access mutations; use the documented IAM endpoint")
	}
	return nil
}

func rejectCurrentIAMIdentityTarget(cmd *cobra.Command, action, targetFlag string) error {
	targetID, err := requiredID(cmd, targetFlag)
	if err != nil {
		return err
	}

	accountsClient, err := createAccountsClient(cmd)
	if err != nil {
		return err
	}
	response, err := accountsClient.RequestWithStatusNoRetrySensitive("GET", "/v1/auth/userinfo", nil, nil)
	if err != nil {
		return fmt.Errorf("get current IAM identity before %s: %w", action, err)
	}
	identityFields, ok := response.Data.(map[string]any)
	if !ok {
		return fmt.Errorf("cannot determine current IAM identity; refusing to %s access", action)
	}
	identityID, comparable, err := currentIdentityIDForTarget(identityFields, targetFlag)
	if err != nil {
		return fmt.Errorf("cannot determine current IAM identity; refusing to %s access: %w", action, err)
	}
	if comparable && identityID == targetID {
		return fmt.Errorf("refusing to %s the current IAM identity", action)
	}
	return nil
}

func currentIdentityIDForTarget(identityFields map[string]any, targetFlag string) (string, bool, error) {
	userType, ok := identityFields["userType"].(string)
	if !ok || userType == "" {
		return "", false, fmt.Errorf("userinfo response has no userType")
	}

	switch targetFlag {
	case "iam-user-id":
		if userType == "user-sa" {
			return "", false, nil
		}
		if userType != "iam-user" {
			return "", false, fmt.Errorf("unsupported userType %q for an IAM user target", userType)
		}
	case "service-account-id":
		if userType == "iam-user" {
			return "", false, nil
		}
		if userType != "user-sa" {
			return "", false, fmt.Errorf("unsupported userType %q for a service account target", userType)
		}
	default:
		return "", false, fmt.Errorf("unsupported IAM target flag %q", targetFlag)
	}

	userID, ok := identityFields["userId"].(string)
	if !ok || userID == "" {
		return "", false, fmt.Errorf("userinfo response has no userId")
	}
	return userID, true, nil
}

func mutationMethod(action string) string {
	if action == "attach" {
		return "POST"
	}
	return "DELETE"
}

func title(value string) string {
	if value == "" {
		return value
	}
	return string(value[0]-'a'+'A') + value[1:]
}
