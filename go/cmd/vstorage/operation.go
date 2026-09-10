package vstorage

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type pathParameter = opengine.PathParam

type queryParameter = opengine.QueryParam

type apiFamily uint8

const (
	commonFamily apiFamily = iota
	hcm03Family
	cephFamily
)

type vstorageExtra struct {
	Family   apiFamily
	PoCField string
}

func extraOf(d opengine.Descriptor) vstorageExtra {
	extra, _ := d.Extra.(vstorageExtra)
	return extra
}

var (
	requiredObjectBody = &opengine.BodyContract{Kind: opengine.ObjectBody, Usage: "Request body as a JSON object"}
	requiredArrayBody  = &opengine.BodyContract{Kind: opengine.ArrayBody, Usage: "Request body as a JSON array"}
	optionalJSONBody   = &opengine.BodyContract{Kind: opengine.OptionalJSONBody, Usage: "Optional request body as JSON (the official schema is not published)"}
	noBody             *opengine.BodyContract
)

type vstorageAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type clientFactory func(*cobra.Command) (vstorageAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (vstorageAPI, error) {
	return cli.NewClient(cmd, "vstorage")
}

var _ vstorageAPI = (*client.GreennodeClient)(nil)

var vstorageSpec = opengine.Spec[vstorageAPI]{
	ServiceName: "vStorage",
	NewClient: func(cmd *cobra.Command, _ opengine.Descriptor) (vstorageAPI, error) {
		return newClient(cmd)
	},
	PostParse: func(cmd *cobra.Command, d opengine.Descriptor, _ string, _ map[string]string, body any) (any, any, error) {
		newBody, err := applyPoC(cmd, d, body)
		return newBody, nil, err
	},
	OfflineValidate: func(cmd *cobra.Command, d opengine.Descriptor) error {
		return validateExplicitRegion(cmd, extraOf(d).Family)
	},
	LiveValidate: func(cmd *cobra.Command, d opengine.Descriptor) error {
		return validateLiveRegion(cmd, extraOf(d).Family)
	},
	Execute: func(_ *cobra.Command, apiClient vstorageAPI, d opengine.Descriptor, path string, query map[string]string, body, _ any) (client.HTTPResponse, error) {
		return execute(apiClient, d.Method, path, query, body)
	},
	ResponseError: func(d opengine.Descriptor, response client.HTTPResponse) error {
		return responseError(response, d)
	},
	AlwaysOutput: true,
	ExtraFlags: func(cmd *cobra.Command, d opengine.Descriptor) {
		if extraOf(d).PoCField != "" {
			cmd.Flags().Bool("poc", false, "Pay with PoC credits through the documented project-creation request")
		}
	},
}

func newOperationCommand(op operation) *cobra.Command {
	return opengine.NewCommand(vstorageSpec, op)
}

func responseError(response client.HTTPResponse, op operation) error {
	if response.Empty {
		if allowsEmptySuccess(op.Method, response.StatusCode) {
			return nil
		}
		return fmt.Errorf("vStorage API returned an empty HTTP %d response for %s %s; expected JSON", response.StatusCode, op.Method, op.Path)
	}

	envelope, ok := response.Data.(map[string]any)
	if !ok {
		return nil
	}
	success, ok := envelope["success"].(bool)
	if !ok || success {
		return nil
	}
	if message, ok := envelope["errorMsg"].(string); ok && strings.TrimSpace(message) != "" {
		return errors.New(message)
	}
	if code, ok := envelope["code"]; ok && code != nil {
		return fmt.Errorf("vStorage API reported failure with code %v", code)
	}
	return errors.New("vStorage API reported failure")
}

func allowsEmptySuccess(method string, statusCode int) bool {
	return statusCode == http.StatusCreated && (method == http.MethodPost || method == http.MethodPut) ||
		statusCode == http.StatusNoContent && method == http.MethodDelete
}

func validateResourceName(resource string) func(value, flag string) error {
	return func(value, flag string) error {
		if err := validateUTF8NoControl(value, flag); err != nil {
			return err
		}
		if len([]rune(value)) < 3 || len([]rune(value)) > 235 {
			return fmt.Errorf("invalid %s: %s names must contain 3 to 235 characters", flag, resource)
		}
		for _, r := range value {
			if !(isASCIIAlphaNumeric(r) || strings.ContainsRune(". _-@", r)) {
				return fmt.Errorf("invalid %s: unsupported %s-name character %q", flag, resource, r)
			}
		}
		return nil
	}
}

func validateObjectName(value, flag string) error {
	return validateUTF8NoControl(value, flag)
}

func validateSwiftObjectName(value, flag string) error {
	if err := validateUTF8NoControl(value, flag); err != nil {
		return err
	}
	if len([]rune(value)) > 255 {
		return fmt.Errorf("invalid %s: object and directory names must not exceed 255 characters", flag)
	}
	return nil
}

func validateLifecycleRuleName(value, flag string) error {
	if err := validateUTF8NoControl(value, flag); err != nil {
		return err
	}
	if len([]rune(value)) < 5 || len([]rune(value)) > 50 {
		return fmt.Errorf("invalid %s: lifecycle rule names must contain 5 to 50 characters", flag)
	}
	for _, r := range value {
		if !(isASCIIAlphaNumeric(r) || strings.ContainsRune(" _-", r)) {
			return fmt.Errorf("invalid %s: unsupported lifecycle-rule-name character %q", flag, r)
		}
	}
	return nil
}

func validateUTF8NoControl(value, flag string) error {
	if value == "" {
		return fmt.Errorf("invalid %s: value must not be empty", flag)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("invalid %s: value must be valid UTF-8", flag)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("invalid %s: control characters are not allowed", flag)
		}
	}
	return nil
}

func isASCIIAlphaNumeric(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

func validateExplicitRegion(cmd *cobra.Command, family apiFamily) error {
	if family == commonFamily || cmd.Flags().Lookup("region") == nil {
		return nil
	}
	if endpoint, _ := cmd.Flags().GetString("endpoint-url"); endpoint != "" {
		return nil
	}
	region, _ := cmd.Flags().GetString("region")
	if region == "" {
		return nil
	}
	return validateFamilyRegion(family, region)
}

func validateLiveRegion(cmd *cobra.Command, family apiFamily) error {
	if family == commonFamily || cmd.Flags().Lookup("region") == nil {
		return nil
	}
	if endpoint, _ := cmd.Flags().GetString("endpoint-url"); endpoint != "" {
		return nil
	}
	region, _ := cmd.Flags().GetString("region")
	if region == "" {
		profile, _ := cmd.Flags().GetString("profile")
		cfg, err := config.LoadConfig(profile)
		if err != nil {
			return err
		}
		region = cfg.Region
	}
	return validateFamilyRegion(family, region)
}

func validateFamilyRegion(family apiFamily, region string) error {
	switch family {
	case hcm03Family:
		if region != "HCM-3" {
			return fmt.Errorf("hcm03 container commands require region HCM-3, got %q", region)
		}
	case cephFamily:
		if region != "HAN" && region != "HCM-4" {
			return fmt.Errorf("ceph bucket commands require region HAN or HCM-4, got %q", region)
		}
	}
	return nil
}

func applyPoC(cmd *cobra.Command, op operation, body any) (any, error) {
	extra := extraOf(op)
	if extra.PoCField == "" || !cmd.Flags().Changed("poc") {
		return body, nil
	}
	enabled, _ := cmd.Flags().GetBool("poc")
	if !enabled {
		return body, nil
	}
	object, ok := body.(map[string]any)
	if !ok {
		return nil, errors.New("--poc requires a JSON object request body")
	}
	if value, present := object[extra.PoCField]; present {
		selected, ok := value.(bool)
		if !ok || !selected {
			return nil, fmt.Errorf("--poc conflicts with body field %q", extra.PoCField)
		}
	}
	object[extra.PoCField] = true
	return object, nil
}

func execute(apiClient vstorageAPI, method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		return apiClient.RequestWithStatus(method, path, query, body)
	default:
		return client.HTTPResponse{}, fmt.Errorf("unsupported vStorage method %q", method)
	}
}
