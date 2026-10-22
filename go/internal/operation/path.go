package operation

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

// BuildPath validates bound identifiers and escapes them into URL placeholders.
func BuildPath(cmd *cobra.Command, serviceName, template string, params []PathParam) (string, error) {
	path := template
	for _, parameter := range params {
		value, _ := cmd.Flags().GetString(parameter.Flag)
		validate := parameter.Validate
		if validate == nil {
			validate = validator.ValidateID
		}
		if err := validate(value, parameter.Flag); err != nil {
			return "", err
		}
		path = strings.ReplaceAll(path, "{"+parameter.Placeholder+"}", url.PathEscape(value))
	}
	if strings.ContainsAny(path, "{}") {
		return "", fmt.Errorf("internal %s contract error: unresolved path parameter in %s", serviceName, template)
	}
	return path, nil
}

// SecretPathValues returns declared credentials for preview, prompt, and debug masking.
func SecretPathValues(cmd *cobra.Command, params []PathParam) []string {
	var secrets []string
	for _, parameter := range params {
		if !parameter.Secret {
			continue
		}
		if value, _ := cmd.Flags().GetString(parameter.Flag); value != "" {
			secrets = append(secrets, value)
		}
	}
	return secrets
}
