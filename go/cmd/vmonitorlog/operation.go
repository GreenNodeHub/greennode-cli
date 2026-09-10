package vmonitorlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	opengine "github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

type operation = opengine.Descriptor

type queryParameter = opengine.QueryParam

var requiredObjectBody = &opengine.BodyContract{Kind: opengine.ObjectBody, Usage: "Request body as a JSON object"}

type vmonitorLogAPI interface {
	RequestWithStatus(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestWithStatusNoRetrySensitive(string, string, map[string]string, any) (client.HTTPResponse, error)
	RequestBytes(string, string, map[string]string, []byte, string) (client.BytesResponse, error)
}

type clientFactory func(*cobra.Command) (vmonitorLogAPI, error)

var newClient clientFactory = func(cmd *cobra.Command) (vmonitorLogAPI, error) {
	return cli.NewClientWithEndpoint(cmd, endpoint)
}

var _ vmonitorLogAPI = (*client.GreennodeClient)(nil)

var vmonitorLogSpec = opengine.Spec[vmonitorLogAPI]{
	ServiceName: "vMonitor Log",
	NewClient: func(cmd *cobra.Command, _ opengine.Descriptor) (vmonitorLogAPI, error) {
		return newClient(cmd)
	},
	DryRunFields: func(cmd *cobra.Command, d opengine.Descriptor, path string, query map[string]string, body any, _ any) (map[string]any, error) {
		fields := opengine.DefaultDryRunFields(d, path, query, body)
		if d.Download {
			outputFile, _ := cmd.Flags().GetString("output-file")
			if outputFile == "" {
				return nil, errors.New("output-file must not be empty")
			}
			fields["output_file"] = outputFile
		}
		return fields, nil
	},
	RunDownload: runDownload,
	Execute: func(_ *cobra.Command, apiClient vmonitorLogAPI, d opengine.Descriptor, path string, query map[string]string, body, _ any) (client.HTTPResponse, error) {
		if d.SecretResponse {
			return apiClient.RequestWithStatusNoRetrySensitive(d.Method, path, query, body)
		}
		return apiClient.RequestWithStatus(d.Method, path, query, body)
	},
	ResponseError: func(d opengine.Descriptor, response client.HTTPResponse) error {
		return opengine.EmptyResponseError("vMonitor Log", d, response)
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
		return cli.RedactJSON(data), nil
	},
}

func newOperationCommand(op operation) *cobra.Command {
	names, err := pathParameters(op.Path)
	if err != nil {
		panic(fmt.Sprintf("invalid vMonitor Log operation path %q: %v", op.Path, err))
	}
	op.Paths = make([]opengine.PathParam, len(names))
	for i, name := range names {
		flag := flagName(name)
		op.Paths[i] = opengine.PathParam{Placeholder: name, Flag: flag, Usage: "Path parameter " + name, Validate: pathValidatorFor(flag)}
	}
	return opengine.NewCommand(vmonitorLogSpec, op)
}

func pathValidatorFor(flag string) func(value, name string) error {
	switch flag {
	case "cdn-domain":
		return validateDomain
	case "bucket-name":
		return validateStorageName
	default:
		return nil
	}
}

func runDownload(cmd *cobra.Command, op operation, path string, query map[string]string) error {
	outputFile, exists, err := downloadOutput(cmd)
	if err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")
	if exists && !cli.Confirm(force, fmt.Sprintf("Overwrite existing output file %s?", outputFile)) {
		return cli.ConfirmationError()
	}
	apiClient, err := newClient(cmd)
	if err != nil {
		return err
	}
	response, err := apiClient.RequestBytes(op.Method, path, query, nil, "application/json")
	if err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return errors.New("vMonitor Log certificate download returned an empty successful response")
	}
	if err := writeProtectedFile(outputFile, response.Data, !exists); err != nil {
		return fmt.Errorf("write certificate bundle to %s: %w", outputFile, err)
	}
	return cli.Output(cmd, map[string]any{"output_file": outputFile, "bytes": len(response.Data), "content_type": response.ContentType})
}

func writeProtectedFile(path string, data []byte, exclusive bool) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("output-file must be a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".grn-certificate-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if !exclusive {
		return os.Rename(file.Name(), path)
	}
	if err := os.Link(file.Name(), path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("output file was created while the download was in progress; refusing to overwrite")
		}
		return err
	}
	return nil
}

func downloadOutput(cmd *cobra.Command) (string, bool, error) {
	outputFile, _ := cmd.Flags().GetString("output-file")
	if outputFile == "" {
		return "", false, errors.New("output-file must not be empty")
	}
	info, err := os.Lstat(outputFile)
	switch {
	case err == nil && !info.Mode().IsRegular():
		return "", false, fmt.Errorf("output-file %s must be a regular file", outputFile)
	case err == nil:
		return outputFile, true, nil
	case !os.IsNotExist(err):
		return "", false, fmt.Errorf("inspect output-file %s: %w", outputFile, err)
	}
	if dir := filepath.Dir(outputFile); dir != "." {
		info, err := os.Stat(dir)
		if err != nil {
			return "", false, fmt.Errorf("inspect output directory %s: %w", dir, err)
		}
		if !info.IsDir() {
			return "", false, fmt.Errorf("output directory %s is not a directory", dir)
		}
	}
	return outputFile, false, nil
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
		if strings.Contains(remaining[:start], "}") {
			return nil, errors.New("unmatched closing brace")
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

func validateDomain(value, name string) error {
	if len(value) > 253 || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return fmt.Errorf("invalid %s: %q must be a DNS domain name", name, value)
	}
	labels := strings.Split(value, ".")
	if len(labels) < 2 {
		return fmt.Errorf("invalid %s: %q must be a DNS domain name", name, value)
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("invalid %s: %q must be a DNS domain name", name, value)
		}
		for _, character := range label {
			if !isASCIIAlphaNumeric(character) && character != '-' {
				return fmt.Errorf("invalid %s: %q must be a DNS domain name", name, value)
			}
		}
	}
	return nil
}

func validateStorageName(value, name string) error {
	if value == "" || !utf8.ValidString(value) {
		return fmt.Errorf("invalid %s: value must be non-empty UTF-8", name)
	}
	for _, character := range value {
		if unicode.IsControl(character) || !(isASCIIAlphaNumeric(character) || strings.ContainsRune(". _-@", character)) {
			return fmt.Errorf("invalid %s: unsupported character %q", name, character)
		}
	}
	return nil
}

func isASCIIAlphaNumeric(character rune) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

func queryFlagName(parameter queryParameter) string {
	return parameter.Flag
}

func flagName(name string) string {
	return opengine.FlagName(name)
}
