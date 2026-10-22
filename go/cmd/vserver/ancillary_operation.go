package vserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

const configuredProjectPlaceholder = "<configured-project-id>"

type ancillaryQueryKind uint8

const (
	ancillaryQueryString ancillaryQueryKind = iota
	ancillaryQueryInteger
	ancillaryQueryBoolean
)

type ancillaryQueryParameter struct {
	Name     string
	Flag     string
	Required bool
	Kind     ancillaryQueryKind
}

type ancillaryPathKind uint8

const (
	ancillaryPathString ancillaryPathKind = iota
	ancillaryPathInt32
)

type ancillaryPathParameter struct {
	Name string
	Kind ancillaryPathKind
}

type ancillaryBodyContract struct {
	Name           string
	RequiredFields []string
}

type ancillaryOperation struct {
	Parents       []string
	Use           string
	Short         string
	Tag           string
	OperationID   string
	Method        string
	Path          string
	ProjectScope  string
	PathKinds     map[string]ancillaryPathKind
	Queries       []ancillaryQueryParameter
	Body          *ancillaryBodyContract
	SuccessStatus int
	ResponseBody  bool
	Mutation      bool
	Destructive   bool
}

type ancillaryAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetry(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type ancillaryClientFactory func(*cobra.Command) (ancillaryAPI, *config.Config, error)

var newAncillaryClient ancillaryClientFactory = func(cmd *cobra.Command) (ancillaryAPI, *config.Config, error) {
	return vserverclient.BuildOperationClient(cmd, cmd.Annotations["vserver-portal-user-id"] != "false")
}

var _ ancillaryAPI = (*client.GreennodeClient)(nil)

func newAncillaryOperationCommand(op ancillaryOperation) *cobra.Command {
	parameters, err := ancillaryPathParameters(op.Path, op.PathKinds)
	if err != nil {
		panic(fmt.Sprintf("invalid vServer operation path %q: %v", op.Path, err))
	}
	cmd := &cobra.Command{
		Use:   op.Use,
		Short: op.Short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := buildAncillaryPath(cmd, op, parameters, configuredProjectPlaceholder)
			if err != nil {
				return err
			}
			query, err := buildAncillaryQuery(cmd, op.Queries)
			if err != nil {
				return err
			}
			body, err := parseAncillaryBody(cmd, op.Body)
			if err != nil {
				return err
			}
			if err := validateAncillaryBodyTarget(cmd, op, body); err != nil {
				return err
			}

			if op.Mutation {
				dryRun, _ := cmd.Flags().GetBool("dry-run")
				if dryRun {
					cli.PrintDryRun(ancillaryVerb(op.Use), "vServer request", ancillaryDryRunFields(op, path, query, body))
					return nil
				}
			}
			apiClient, cfg, err := newAncillaryClient(cmd)
			if err != nil {
				return err
			}
			if ancillaryHasProjectParameter(op, parameters) {
				projectID, err := vserverclient.ProjectID(cfg)
				if err != nil {
					return err
				}
				path, err = buildAncillaryPath(cmd, op, parameters, projectID)
				if err != nil {
					return err
				}
			}
			if op.Destructive {
				force, _ := cmd.Flags().GetBool("force")
				if !cli.Confirm(force, fmt.Sprintf("Proceed with %s %s?", op.Method, path)) {
					return cli.ConfirmationError()
				}
			}

			response, err := executeAncillary(apiClient, op, path, query, body)
			if err != nil {
				return err
			}
			if err := ancillaryResponseError(op, response); err != nil {
				return err
			}
			if response.Empty {
				return nil
			}
			return vserverclient.Output(cmd, cfg, response.Data)
		},
	}
	if op.Tag == "Market-place" || op.Tag == "Protocol" {
		cmd.Annotations = map[string]string{"vserver-portal-user-id": "false"}
	}

	for _, parameter := range parameters {
		if ancillaryProjectParameter(op, parameter.Name) {
			continue
		}
		flag := ancillaryFlagName(parameter.Name)
		cmd.Flags().String(flag, "", "Path parameter "+parameter.Name)
		_ = cmd.MarkFlagRequired(flag)
	}
	for _, parameter := range op.Queries {
		flag := parameter.Flag
		if flag == "" {
			flag = ancillaryFlagName(parameter.Name)
		}
		cmd.Flags().String(flag, "", "Query parameter "+parameter.Name)
		if parameter.Required {
			_ = cmd.MarkFlagRequired(flag)
		}
	}
	if op.Body != nil {
		cmd.Flags().String("body", "", "Request body as a "+op.Body.Name+" JSON object")
		_ = cmd.MarkFlagRequired("body")
	}
	if op.Mutation {
		cmd.Flags().Bool("dry-run", false, "Preview the validated request without reading configuration or calling the API")
	}
	if op.Destructive {
		cmd.Flags().Bool("force", false, "Skip the destructive-action confirmation")
	}
	return cmd
}

func executeAncillary(apiClient ancillaryAPI, op ancillaryOperation, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	if op.Mutation {
		return apiClient.RequestWithStatusNoRetry(op.Method, path, query, body)
	}
	return apiClient.RequestWithStatus(op.Method, path, query, body)
}

func ancillaryPathParameters(path string, kinds map[string]ancillaryPathKind) ([]ancillaryPathParameter, error) {
	var parameters []ancillaryPathParameter
	seen := make(map[string]bool)
	remaining := path
	for remaining != "" {
		start := strings.IndexByte(remaining, '{')
		if start < 0 {
			if strings.Contains(remaining, "}") {
				return nil, errors.New("unmatched closing brace")
			}
			break
		}
		end := strings.IndexByte(remaining[start+1:], '}')
		if end < 0 {
			return nil, errors.New("unclosed path parameter")
		}
		end += start + 1
		name := remaining[start+1 : end]
		if name == "" || strings.ContainsAny(name, "{}") {
			return nil, errors.New("invalid path parameter")
		}
		kind := ancillaryPathString
		if configuredKind, ok := kinds[name]; ok {
			kind = configuredKind
		}
		parameters = append(parameters, ancillaryPathParameter{Name: name, Kind: kind})
		seen[name] = true
		remaining = remaining[end+1:]
	}
	for name := range kinds {
		if !seen[name] {
			return nil, fmt.Errorf("path kind configured for absent parameter %q", name)
		}
	}
	return parameters, nil
}

func buildAncillaryPath(cmd *cobra.Command, op ancillaryOperation, parameters []ancillaryPathParameter, projectID string) (string, error) {
	template := op.Path
	path := template
	for _, parameter := range parameters {
		value := projectID
		escaped := ""
		if !ancillaryProjectParameter(op, parameter.Name) {
			flag := ancillaryFlagName(parameter.Name)
			value, _ = cmd.Flags().GetString(flag)
			switch parameter.Kind {
			case ancillaryPathString:
				if err := validator.ValidateID(value, flag); err != nil {
					return "", err
				}
			case ancillaryPathInt32:
				parsed, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					return "", fmt.Errorf("invalid %s: must be a 32-bit integer", flag)
				}
				value = strconv.FormatInt(parsed, 10)
			default:
				return "", fmt.Errorf("internal vServer contract error: unsupported path kind for %s", parameter.Name)
			}
			escaped = url.PathEscape(value)
		} else if value == configuredProjectPlaceholder {
			escaped = value
		} else {
			escaped = url.PathEscape(value)
		}
		path = strings.ReplaceAll(path, "{"+parameter.Name+"}", escaped)
	}
	if strings.ContainsAny(path, "{}") {
		return "", fmt.Errorf("internal vServer contract error: unresolved path parameter in %s", template)
	}
	return path, nil
}

func buildAncillaryQuery(cmd *cobra.Command, parameters []ancillaryQueryParameter) (map[string]string, error) {
	if len(parameters) == 0 {
		return nil, nil
	}
	query := make(map[string]string)
	for _, parameter := range parameters {
		flag := parameter.Flag
		if flag == "" {
			flag = ancillaryFlagName(parameter.Name)
		}
		value, _ := cmd.Flags().GetString(flag)
		if value == "" {
			if parameter.Required {
				return nil, fmt.Errorf("%s must not be empty", flag)
			}
			continue
		}
		switch parameter.Kind {
		case ancillaryQueryInteger:
			if _, err := strconv.ParseInt(value, 10, 32); err != nil {
				return nil, fmt.Errorf("invalid %s: must be a 32-bit integer", flag)
			}
		case ancillaryQueryBoolean:
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("invalid %s: must be true or false", flag)
			}
			value = strconv.FormatBool(parsed)
		}
		query[parameter.Name] = value
	}
	if len(query) == 0 {
		return nil, nil
	}
	return query, nil
}

func parseAncillaryBody(cmd *cobra.Command, contract *ancillaryBodyContract) (any, error) {
	if contract == nil {
		return nil, nil
	}
	raw, _ := cmd.Flags().GetString("body")
	if raw == "" {
		return nil, errors.New("body must not be empty")
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("invalid body JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("invalid body JSON: multiple JSON values are not allowed")
		}
		return nil, fmt.Errorf("invalid body JSON: %w", err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("body must be a JSON object")
	}
	for _, field := range contract.RequiredFields {
		if fieldValue, present := object[field]; !present || fieldValue == nil {
			return nil, fmt.Errorf("body is missing required field %q for %s", field, contract.Name)
		}
	}
	return object, nil
}

func validateAncillaryBodyTarget(cmd *cobra.Command, op ancillaryOperation, body any) error {
	object, ok := body.(map[string]any)
	if !ok {
		return nil
	}
	bind := func(field, flag string) error {
		value, present := object[field]
		if !present {
			return nil
		}
		bodyID, ok := value.(string)
		if !ok {
			return fmt.Errorf("body field %q must be a string matching --%s", field, flag)
		}
		flagID, _ := cmd.Flags().GetString(flag)
		if bodyID != flagID {
			return fmt.Errorf("body field %q must match --%s", field, flag)
		}
		return nil
	}

	switch op.OperationID {
	case "updateRulesUsingPUT":
		return bind("aclId", "acl-id")
	case "updateAssociatedSubnetsUsingPUT":
		return bind("aclId", "uuid")
	case "deletePersistentVolumeUsingDELETE":
		return bind("persistentVolumeId", "pv-id")
	default:
		return nil
	}
}

func ancillaryResponseError(op ancillaryOperation, response client.HTTPResponse) error {
	if response.StatusCode != op.SuccessStatus {
		return fmt.Errorf("vServer API returned unexpected HTTP %d for %s %s; expected HTTP %d", response.StatusCode, op.Method, op.Path, op.SuccessStatus)
	}
	if op.ResponseBody && response.Empty {
		return fmt.Errorf("vServer API returned an empty HTTP %d response for %s %s; expected JSON", response.StatusCode, op.Method, op.Path)
	}
	if !op.ResponseBody && !response.Empty {
		return fmt.Errorf("vServer API returned a JSON body for %s %s; documented HTTP %d response is empty", op.Method, op.Path, response.StatusCode)
	}
	return nil
}

func ancillaryDryRunFields(op ancillaryOperation, path string, query map[string]string, body any) map[string]any {
	fields := map[string]any{
		"headers": map[string]any{"portal-user-id": "<configured>"},
		"method":  op.Method,
		"path":    path,
	}
	if len(query) > 0 {
		fields["query"] = query
	}
	if body != nil {
		fields["body"] = cli.RedactJSON(body)
	}
	return fields
}

func ancillaryHasProjectParameter(op ancillaryOperation, parameters []ancillaryPathParameter) bool {
	for _, parameter := range parameters {
		if ancillaryProjectParameter(op, parameter.Name) {
			return true
		}
	}
	return false
}

func ancillaryProjectParameter(op ancillaryOperation, parameter string) bool {
	return op.ProjectScope == parameter
}

func ancillaryFlagName(name string) string {
	var flag strings.Builder
	for index, character := range name {
		switch {
		case character == '_':
			flag.WriteByte('-')
		case unicode.IsUpper(character):
			if index > 0 {
				flag.WriteByte('-')
			}
			flag.WriteRune(unicode.ToLower(character))
		default:
			flag.WriteRune(character)
		}
	}
	return flag.String()
}

func ancillaryVerb(use string) string {
	if before, _, ok := strings.Cut(use, "-"); ok {
		return before
	}
	return use
}

func registerAncillaryOperations(root *cobra.Command) {
	groups := make(map[string]*cobra.Command)
	for _, command := range root.Commands() {
		groups[command.Name()] = command
	}
	for _, op := range ancillaryOperations() {
		parent := root
		var route []string
		for _, name := range op.Parents {
			route = append(route, name)
			key := strings.Join(route, " ")
			group := groups[key]
			if group == nil {
				group = newAncillaryGroup(name)
				parent.AddCommand(group)
				groups[key] = group
			}
			parent = group
		}
		parent.AddCommand(newAncillaryOperationCommand(op))
	}
}

func newAncillaryGroup(use string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: "Manage vServer " + strings.ReplaceAll(use, "-", " ") + " resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}
