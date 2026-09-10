package vdb

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/client"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/greennodehub/greennode-cli/internal/redact"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type queryParameter = opengine.QueryParam

const (
	queryString  = opengine.QueryString
	queryInteger = opengine.QueryInteger
	queryBoolean = opengine.QueryBoolean
	queryObject  = opengine.QueryObject

	bodyObject = opengine.ObjectBody
	bodyArray  = opengine.ArrayBody
)

func objectBody(name string) *opengine.BodyContract {
	return &opengine.BodyContract{Kind: bodyObject, Style: opengine.BodyStyleJSON, Name: name, Usage: fmt.Sprintf("Request body as a %s JSON object", name)}
}

func arrayBody(name string, requiredItemFields ...string) *opengine.BodyContract {
	return &opengine.BodyContract{
		Kind:               bodyArray,
		Style:              opengine.BodyStyleJSON,
		Name:               name,
		Usage:              fmt.Sprintf("Request body as a %s JSON array", name),
		RequiredItemFields: requiredItemFields,
		StrictArrayItems:   true,
	}
}

type pocLocation uint8

const (
	pocDirectBody pocLocation = iota
	pocDatabaseInstanceConfig
)

type pocContract struct {
	Field    string
	Location pocLocation
}

type vdbAPI interface {
	SetHeader(string, string)
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetry(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetrySensitive(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetryRaw(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetrySensitiveRaw(string, string, map[string]string, any) (client.HTTPResponse, error)
}

type clientFactory func(*cobra.Command, bool) (vdbAPI, error)

var newClient clientFactory = func(cmd *cobra.Command, requirePortalUserID bool) (vdbAPI, error) {
	return vdbclient.BuildClient(cmd, requirePortalUserID)
}

var _ vdbAPI = (*client.GreennodeClient)(nil)

var vdbSpec = opengine.Spec[vdbAPI]{
	ServiceName: "vDB",
	NewClient: func(cmd *cobra.Command, d opengine.Descriptor) (vdbAPI, error) {
		return newClient(cmd, !d.NoPortalUserID)
	},
	PostParse: func(cmd *cobra.Command, d opengine.Descriptor, _ string, _ map[string]string, body any) (any, any, error) {
		body, pocEnabled, err := applyPoC(cmd, d, body)
		if err != nil {
			return nil, nil, err
		}
		if err := validateBodyContract(cmd, d, body); err != nil {
			return nil, nil, err
		}
		userType, err := commandUserType(cmd, d.UserType)
		if err != nil {
			return nil, nil, err
		}
		if pocEnabled {
			if !d.UserType {
				return nil, nil, fmt.Errorf("internal vDB contract error: --poc requires a documented user-type header")
			}
			if userType == "ROOT_USER" {
				return nil, nil, errors.New("--poc requires automatic payment; omit --user-type or set --user-type IAM_USER")
			}
			userType = "IAM_USER"
		}
		return body, userType, nil
	},
	DryRunUsage: "Preview the validated request without reading configuration or calling the API",
	DryRunFields: func(_ *cobra.Command, d opengine.Descriptor, path string, query map[string]string, body any, state any) (map[string]any, error) {
		userType, _ := state.(string)
		return dryRunFields(d, path, query, body, userType), nil
	},
	Execute: func(_ *cobra.Command, apiClient vdbAPI, d opengine.Descriptor, path string, query map[string]string, body, state any) (client.HTTPResponse, error) {
		if userType, _ := state.(string); userType != "" {
			apiClient.SetHeader("user-type", userType)
		}
		return execute(apiClient, d, path, query, body)
	},
	ResponseError: func(d opengine.Descriptor, response client.HTTPResponse) error {
		return responseError(d, response)
	},
	TransformOutput: func(cmd *cobra.Command, d opengine.Descriptor, data any) (any, error) {
		if !d.SecretResponse {
			return data, nil
		}
		showSecret, _ := cmd.Flags().GetBool("show-secret")
		if showSecret {
			return data, nil
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Credential response redacted. Re-run with --show-secret only when you are ready to handle it securely.")
		return redactSecretResponse(data), nil
	},
	ExtraFlags: func(cmd *cobra.Command, d opengine.Descriptor) {
		if d.UserType {
			cmd.Flags().String("user-type", "", "Payment flow identity: ROOT_USER or IAM_USER")
		}
		if poc, ok := d.Extra.(*pocContract); ok && poc != nil {
			cmd.Flags().Bool("poc", false, "Pay with PoC credits through the documented automatic-payment flow")
		}
	},
}

func newOperationCommand(op operation) *cobra.Command {
	names, err := pathParameters(op.Path)
	if err != nil {
		panic(fmt.Sprintf("invalid vDB operation path %q: %v", op.Path, err))
	}
	op.Paths = make([]opengine.PathParam, len(names))
	for i, name := range names {
		op.Paths[i] = opengine.PathParam{Placeholder: name, Flag: flagName(name), Usage: "Path parameter " + name}
	}
	if len(op.Queries) > 0 {
		queries := make([]opengine.QueryParam, len(op.Queries))
		copy(queries, op.Queries)
		for i := range queries {
			if queries[i].Flag == "" {
				queries[i].Flag = flagName(queries[i].WireName)
			}
			if queries[i].Usage == "" {
				queries[i].Usage = "Query parameter " + queries[i].WireName
			}
			if queries[i].Kind == opengine.QueryObject {
				queries[i].ObjectStyle = opengine.QueryObjectJSON
				queries[i].EncodeError = func(flag string, err error) error {
					return fmt.Errorf("encode %s: %w", flag, err)
				}
			}
		}
		op.Queries = queries
	}
	return opengine.NewCommand(vdbSpec, op)
}

func execute(apiClient vdbAPI, op operation, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	if op.SecretResponse && op.RawResponse {
		return apiClient.RequestWithStatusNoRetrySensitiveRaw(op.Method, path, query, body)
	}
	if op.SecretResponse {
		return apiClient.RequestWithStatusNoRetrySensitive(op.Method, path, query, body)
	}
	if op.RawResponse {
		return apiClient.RequestWithStatusNoRetryRaw(op.Method, path, query, body)
	}
	if op.Mutation {
		return apiClient.RequestWithStatusNoRetry(op.Method, path, query, body)
	}
	return apiClient.RequestWithStatus(op.Method, path, query, body)
}

func pathParameters(path string) ([]string, error) {
	var parameters []string
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
		parameters = append(parameters, name)
		remaining = remaining[end+1:]
	}
	return parameters, nil
}

func applyPoC(cmd *cobra.Command, op operation, body any) (any, bool, error) {
	poc, _ := op.Extra.(*pocContract)
	if poc == nil || !cmd.Flags().Changed("poc") {
		return body, false, nil
	}
	enabled, _ := cmd.Flags().GetBool("poc")
	if !enabled {
		return body, false, nil
	}

	object, ok := body.(map[string]any)
	if !ok {
		return nil, false, errors.New("--poc requires a JSON object request body")
	}
	switch poc.Location {
	case pocDirectBody:
		if err := setPoCField(object, poc.Field); err != nil {
			return nil, false, err
		}
	case pocDatabaseInstanceConfig:
		instances, present := object["databaseInstances"]
		if !present {
			return nil, false, errors.New("--poc requires body.databaseInstances")
		}
		items, ok := instances.([]any)
		if !ok || len(items) == 0 {
			return nil, false, errors.New("--poc requires body.databaseInstances to be a non-empty JSON array")
		}
		for index, item := range items {
			instance, ok := item.(map[string]any)
			if !ok {
				return nil, false, fmt.Errorf("--poc requires body.databaseInstances[%d] to be a JSON object", index)
			}
			config, ok := instance["config"].(map[string]any)
			if !ok {
				return nil, false, fmt.Errorf("--poc requires body.databaseInstances[%d].config to be a JSON object", index)
			}
			if err := setPoCField(config, poc.Field); err != nil {
				return nil, false, err
			}
		}
	default:
		return nil, false, fmt.Errorf("internal vDB contract error: unsupported PoC location %d", poc.Location)
	}
	return body, true, nil
}

func setPoCField(object map[string]any, field string) error {
	if value, present := object[field]; present {
		selected, ok := value.(bool)
		if !ok || !selected {
			return fmt.Errorf("--poc conflicts with body field %q", field)
		}
	}
	object[field] = true
	return nil
}

func commandUserType(cmd *cobra.Command, supported bool) (string, error) {
	if !supported {
		return "", nil
	}
	value, _ := cmd.Flags().GetString("user-type")
	if value == "" || value == "ROOT_USER" || value == "IAM_USER" {
		return value, nil
	}
	return "", fmt.Errorf("invalid user-type %q: must be ROOT_USER or IAM_USER", value)
}

func validateBodyContract(cmd *cobra.Command, op operation, body any) error {
	if items, ok := body.([]any); ok && strings.Contains(op.Path, "/backups/{backupId}/delete") {
		return validateArrayIDs(cmd, items, "backupId", "backup-id")
	}
	object, ok := body.(map[string]any)
	if !ok {
		return nil
	}
	if expected := expectedAction(op.Path); expected != "" {
		if value, present := object["action"]; present && value != expected {
			return fmt.Errorf("body field %q must be %q for %s", "action", expected, op.Use)
		}
	}

	bindDirect := func(field, flag string) error {
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

	switch {
	case strings.Contains(op.Path, "/{instanceId}/create-replicas"):
		return bindDirect("replicaSourceId", "instance-id")
	case strings.Contains(op.Path, "/{dbInstanceId}/create-replicas"):
		return bindDirect("replicaSourceId", "db-instance-id")
	case strings.Contains(op.Path, "/{instanceId}/update/"):
		return bindDirect("dbInstanceId", "instance-id")
	case strings.Contains(op.Path, "/{dbInstanceId}/update-"):
		return bindDirect("dbInstanceId", "db-instance-id")
	case strings.Contains(op.Path, "/backups/{id}/restore"):
		return validateNestedConfigIDs(cmd, object, "backupId", "id")
	case strings.Contains(op.Path, "/backups/{backupId}/restore"):
		return validateNestedConfigIDs(cmd, object, "backupId", "backup-id")
	case strings.Contains(op.Path, "/database-instances/{instanceId}/"):
		return validateObjectArrayIDs(cmd, object, "databaseInstances", "instancesId", "instance-id")
	case strings.Contains(op.Path, "/database-instances/{dbInstanceId}/"):
		return validateObjectArrayIDs(cmd, object, "databaseInstances", "instancesId", "db-instance-id")
	}
	return nil
}

func validateObjectArrayIDs(cmd *cobra.Command, body map[string]any, arrayField, idField, flag string) error {
	value, present := body[arrayField]
	if !present {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return fmt.Errorf("body field %q must be an array", arrayField)
	}
	return validateArrayIDs(cmd, items, idField, flag)
}

func validateArrayIDs(cmd *cobra.Command, items []any, idField, flag string) error {
	want, _ := cmd.Flags().GetString(flag)
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		value, present := object[idField]
		if !present {
			continue
		}
		got, ok := value.(string)
		if !ok || got != want {
			return fmt.Errorf("body item %d field %q must match --%s", index, idField, flag)
		}
	}
	return nil
}

func validateNestedConfigIDs(cmd *cobra.Command, body map[string]any, idField, flag string) error {
	value, present := body["databaseInstances"]
	if !present {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return errors.New("body field \"databaseInstances\" must be an array")
	}
	want, _ := cmd.Flags().GetString(flag)
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		configValue, present := object["config"]
		if !present {
			continue
		}
		configObject, ok := configValue.(map[string]any)
		if !ok {
			return fmt.Errorf("body item %d field %q must be an object", index, "config")
		}
		value, present := configObject[idField]
		if !present {
			continue
		}
		got, ok := value.(string)
		if !ok || got != want {
			return fmt.Errorf("body item %d config field %q must match --%s", index, idField, flag)
		}
	}
	return nil
}

func expectedAction(path string) string {
	switch {
	case strings.Contains(path, "/restore"):
		return "restore_backup"
	case strings.Contains(path, "/actions/delete"), strings.HasSuffix(path, "/delete"):
		return "delete"
	case strings.Contains(path, "/actions/resize"), strings.Contains(path, "/resize-"):
		return "resize"
	case strings.HasSuffix(path, "/detach-replica"):
		return "detach_replica"
	case strings.HasSuffix(path, "/reboot"):
		return "reboot"
	case strings.HasSuffix(path, "/shutdown"):
		return "stop"
	case strings.HasSuffix(path, "/start"):
		return "start"
	default:
		return ""
	}
}

func responseError(op operation, response client.HTTPResponse) error {
	switch response.StatusCode {
	case http.StatusOK:
		if response.Empty && !op.RawResponse {
			return fmt.Errorf("vDB API returned an empty HTTP 200 response for %s %s; expected JSON", op.Method, op.Path)
		}
	case http.StatusAccepted, http.StatusNoContent:
		if !response.Empty {
			return fmt.Errorf("vDB API returned a JSON body for %s %s; documented HTTP %d response is empty", op.Method, op.Path, response.StatusCode)
		}
	default:
		return fmt.Errorf("vDB API returned unexpected HTTP %d for %s %s; expected HTTP 200, 202, or 204", response.StatusCode, op.Method, op.Path)
	}
	return nil
}

func dryRunFields(op operation, path string, query map[string]string, body any, userType string) map[string]any {
	fields := opengine.DefaultDryRunFields(op, path, query, body)
	headers := make(map[string]any)
	if !op.NoPortalUserID {
		headers["portal-user-id"] = "<configured>"
	}
	if userType != "" {
		headers["user-type"] = userType
	}
	if len(headers) > 0 {
		fields["headers"] = headers
	}
	return fields
}

func redactSecretResponse(value any) any {
	if _, ok := value.(string); ok {
		return redact.Value
	}
	if values, ok := value.([]any); ok {
		redacted := make([]any, len(values))
		for index := range values {
			redacted[index] = redact.Value
		}
		return redacted
	}
	return redact.JSON(value)
}

func flagName(name string) string {
	return opengine.FlagName(name)
}
