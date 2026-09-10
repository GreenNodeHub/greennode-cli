package iam

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

type iamMutationPath struct {
	Placeholder string
	Flag        string
	Usage       string
}

type iamMutationBodyShape uint8

const (
	iamMutationBodyObject iamMutationBodyShape = iota
	iamMutationBodyArray
)

type iamMutationBody struct {
	Shape          iamMutationBodyShape
	RequiredFields []string
}

type iamMutationOperation struct {
	Key                 string
	Use                 string
	Short               string
	Method              string
	Path                string
	BuildClient         clientFactory
	Paths               []iamMutationPath
	Body                *iamMutationBody
	Status              int
	ResponseBody        bool
	AllowUnexpectedBody bool
	Destructive         bool
	SecretInput         bool
	SecretResponse      bool
	SelfTargetFlag      string
}

var registeredIAMMutationCommands = map[string]*cobra.Command{}

func mutationCommand(key string) *cobra.Command {
	for _, op := range allIAMMutationOperations() {
		if op.Key == key {
			cmd := newIAMMutationCommand(op)
			registeredIAMMutationCommands[key] = cmd
			return cmd
		}
	}
	panic("missing IAM mutation operation: " + key)
}

func newIAMMutationCommand(op iamMutationOperation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   op.Use,
		Short: op.Short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := buildIAMMutationPath(cmd, op)
			if err != nil {
				return err
			}
			body, err := parseIAMMutationBody(cmd, op)
			if err != nil {
				return err
			}
			if err := rejectIAMMutationEndpointOverride(cmd); err != nil {
				return err
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if dryRun {
				cli.PrintDryRun(mutationVerb(op.Use), "IAM request", iamMutationDryRunFields(op, path, body))
				return nil
			}

			prompt := fmt.Sprintf("Proceed with IAM %s %s?", op.Method, path)
			if op.Destructive {
				prompt += " This action may be irreversible."
			}
			force, _ := cmd.Flags().GetBool("force")
			if !cli.Confirm(force, prompt) {
				return cli.ConfirmationError()
			}
			if op.SelfTargetFlag != "" {
				if err := rejectCurrentIAMIdentityTarget(cmd, mutationVerb(op.Use), op.SelfTargetFlag); err != nil {
					return err
				}
			}

			apiClient, err := op.BuildClient(cmd)
			if err != nil {
				return err
			}
			response, err := executeIAMMutation(apiClient, op, path, body)
			if err != nil {
				return err
			}
			if err := iamMutationResponseError(op, response); err != nil {
				return err
			}
			if response.Empty {
				return nil
			}
			if iamToleratesUnexpectedBody(op, response) && !op.SecretResponse {
				fmt.Fprintln(cmd.ErrOrStderr(), "IAM API returned an undocumented success payload; the response was suppressed because the official contract declares an empty response.")
				return nil
			}

			data := cli.RedactJSON(response.Data)
			if op.SecretResponse {
				showSecret, _ := cmd.Flags().GetBool("show-secret")
				if showSecret {
					data = response.Data
				} else {
					data = redactIAMSecretResponse(response.Data)
					fmt.Fprintln(cmd.ErrOrStderr(), "Secret response redacted. Use --show-secret when you intend to expose returned credentials.")
				}
			}
			return outputResult(cmd, data)
		},
	}

	for _, parameter := range op.Paths {
		addRequiredIDFlag(cmd, parameter.Flag, parameter.Usage)
	}
	if op.Body != nil {
		if op.SecretInput {
			cmd.Flags().String("body-file", "", "Read the secret-bearing request body from a private JSON file")
		} else {
			cmd.Flags().String("body", "", "Request body as a "+iamMutationBodyShapeLabel(op.Body.Shape))
			cmd.Flags().String("body-file", "", "Read the request body from a JSON file")
		}
	}
	cmd.Flags().Bool("dry-run", false, "Preview the validated IAM request without calling the API")
	cmd.Flags().Bool("force", false, "Skip the IAM mutation confirmation")
	if op.SecretResponse {
		cmd.Flags().Bool("show-secret", false, "Print returned secret material in command output")
	}
	return cmd
}

func buildIAMMutationPath(cmd *cobra.Command, op iamMutationOperation) (string, error) {
	path := op.Path
	for _, parameter := range op.Paths {
		value, err := requiredID(cmd, parameter.Flag)
		if err != nil {
			return "", err
		}
		path = strings.ReplaceAll(path, "{"+parameter.Placeholder+"}", url.PathEscape(value))
	}
	if strings.ContainsAny(path, "{}") {
		return "", fmt.Errorf("internal IAM contract error: unresolved path parameter in %s", op.Path)
	}
	return path, nil
}

func parseIAMMutationBody(cmd *cobra.Command, op iamMutationOperation) (any, error) {
	if op.Body == nil {
		return nil, nil
	}

	rawBody := ""
	if cmd.Flags().Lookup("body") != nil {
		rawBody, _ = cmd.Flags().GetString("body")
	}
	bodyFile, _ := cmd.Flags().GetString("body-file")
	if rawBody != "" && bodyFile != "" {
		return nil, errors.New("only one of --body or --body-file may be provided")
	}
	if rawBody == "" && bodyFile == "" {
		return nil, errors.New("one of --body or --body-file must be provided")
	}
	if bodyFile != "" {
		contents, err := readIAMBodyFile(bodyFile, op.SecretInput)
		if err != nil {
			return nil, fmt.Errorf("read --body-file: %w", err)
		}
		rawBody = string(contents)
	}

	body, err := parseIAMJSONBody(rawBody, op.Body.Shape)
	if err != nil {
		return nil, err
	}
	objectBody, isObject := body.(map[string]any)
	if !isObject && len(op.Body.RequiredFields) != 0 {
		return nil, errors.New("internal IAM contract error: required fields need a JSON object body")
	}
	for _, field := range op.Body.RequiredFields {
		if value, ok := objectBody[field]; !ok || value == nil {
			return nil, fmt.Errorf("body is missing required field %q", field)
		}
	}
	return body, nil
}

func parseIAMJSONBody(raw string, shape iamMutationBodyShape) (any, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("invalid JSON request body: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("invalid JSON request body: multiple JSON values are not allowed")
		}
		return nil, fmt.Errorf("invalid JSON request body: %w", err)
	}
	switch shape {
	case iamMutationBodyObject:
		if body, ok := value.(map[string]any); ok {
			return body, nil
		}
	case iamMutationBodyArray:
		if body, ok := value.([]any); ok {
			return body, nil
		}
	default:
		return nil, errors.New("internal IAM contract error: unsupported request body shape")
	}
	return nil, errors.New("request body must be a " + iamMutationBodyShapeLabel(shape))
}

func executeIAMMutation(apiClient *client.GreennodeClient, op iamMutationOperation, path string, body any) (client.HTTPResponse, error) {
	if op.SecretResponse && op.AllowUnexpectedBody {
		response, err := apiClient.RequestWithStatusNoRetrySensitiveRaw(op.Method, path, nil, body)
		if raw, ok := response.Data.(string); ok {
			var value any
			if json.Unmarshal([]byte(raw), &value) == nil {
				response.Data = value
			}
		}
		return response, err
	}
	return apiClient.RequestWithStatusNoRetrySensitive(op.Method, path, nil, body)
}

func readIAMBodyFile(path string, secret bool) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("--body-file must be a regular file, not a symlink")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("--body-file changed while opening")
	}
	if secret && opened.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("--body-file for secret input must not be readable by group or others")
	}
	return io.ReadAll(file)
}

func iamMutationResponseError(op iamMutationOperation, response client.HTTPResponse) error {
	if response.StatusCode != op.Status {
		return fmt.Errorf("IAM API returned HTTP %d for %s %s; expected HTTP %d", response.StatusCode, op.Method, op.Path, op.Status)
	}
	if iamToleratesUnexpectedBody(op, response) {
		return nil
	}
	if response.Empty == op.ResponseBody {
		if op.ResponseBody {
			return fmt.Errorf("IAM API returned an empty HTTP %d response for %s %s; expected JSON", response.StatusCode, op.Method, op.Path)
		}
		return fmt.Errorf("IAM API returned a JSON body for %s %s; the documented HTTP %d response is empty", op.Method, op.Path, op.Status)
	}
	return nil
}

func iamToleratesUnexpectedBody(op iamMutationOperation, response client.HTTPResponse) bool {
	return !response.Empty && !op.ResponseBody && op.AllowUnexpectedBody
}

func iamMutationDryRunFields(op iamMutationOperation, path string, body any) map[string]any {
	fields := map[string]any{"method": op.Method, "path": path}
	if body != nil {
		fields["body"] = client.RedactJSON(body)
		if op.SecretInput {
			fields["body"] = client.RedactedValue
		}
	}
	return fields
}

func iamMutationBodyShapeLabel(shape iamMutationBodyShape) string {
	if shape == iamMutationBodyArray {
		return "JSON array"
	}
	return "JSON object"
}

func redactIAMSecretResponse(value any) any {
	return client.RedactedValue
}

func mutationVerb(use string) string {
	if before, _, ok := strings.Cut(use, "-"); ok {
		return before
	}
	return use
}

func mutationPath(placeholder, flag, usage string) iamMutationPath {
	return iamMutationPath{Placeholder: placeholder, Flag: flag, Usage: usage}
}
