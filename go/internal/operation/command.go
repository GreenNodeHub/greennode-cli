package operation

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/redact"
	"github.com/spf13/cobra"
)

// Spec binds a service client and optional hooks to the shared command pipeline.
type Spec[C any] struct {
	// ServiceName labels generated errors and dry-run previews.
	ServiceName string

	// NewClient constructs the service client for an operation.
	NewClient func(cmd *cobra.Command, d Descriptor) (C, error)

	// PostParse may transform the body and pass per-request state to preview and execution hooks.
	PostParse func(cmd *cobra.Command, d Descriptor, path string, query map[string]string, body any) (newBody any, state any, err error)

	// OfflineValidate runs before dry-run and must not read live configuration.
	OfflineValidate func(cmd *cobra.Command, d Descriptor) error

	// LiveValidate runs after dry-run, before confirmation.
	LiveValidate func(cmd *cobra.Command, d Descriptor) error

	// DryRunUsage overrides the default preview flag description.
	DryRunUsage string

	// DryRunFields overrides DefaultDryRunFields to include service-specific preview data.
	DryRunFields func(cmd *cobra.Command, d Descriptor, path string, query map[string]string, body, state any) (map[string]any, error)

	// RunDownload owns client construction, validation, and output for downloads.
	RunDownload func(cmd *cobra.Command, d Descriptor, path string, query map[string]string) error

	// Execute invokes the service client; required.
	Execute func(cmd *cobra.Command, apiClient C, d Descriptor, path string, query map[string]string, body, state any) (client.HTTPResponse, error)

	// ResponseError checks service-specific success statuses and response shapes.
	ResponseError func(d Descriptor, response client.HTTPResponse) error

	// AlwaysOutput sends empty responses through output formatting.
	AlwaysOutput bool

	// ShowSecretUsage overrides the credential-output flag description.
	ShowSecretUsage string

	// TransformOutput redacts or replaces response data before output.
	TransformOutput func(cmd *cobra.Command, d Descriptor, data any) (any, error)

	// ExtraFlags registers flags beyond the engine's standard set (vdb's
	// --user-type and --poc).
	ExtraFlags func(cmd *cobra.Command, d Descriptor)
}

// secretPathMasker lets the engine require path masking without a concrete client dependency.
type secretPathMasker interface {
	MaskPathValues(values []string)
}

func NewCommand[C any](spec Spec[C], d Descriptor) *cobra.Command {
	// Downloads own their clients, so secret-path downloads require explicit masking support.
	if d.Download {
		for _, parameter := range d.Paths {
			if parameter.Secret {
				panic(fmt.Sprintf("internal %s contract error: download operation %s cannot mask secret path parameter %q", spec.ServiceName, d.Use, parameter.Placeholder))
			}
		}
	}
	cmd := &cobra.Command{
		Use:   d.Use,
		Short: d.Short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := BuildPath(cmd, spec.ServiceName, d.Path, d.Paths)
			if err != nil {
				return err
			}
			// Mask secret paths in previews, prompts, and debug output.
			secretPathValues := SecretPathValues(cmd, d.Paths)
			displayPath := redact.PathValues(path, secretPathValues)
			query, err := BuildQuery(cmd, d.Queries)
			if err != nil {
				return err
			}
			body, err := ParseBody(cmd, d.Body)
			if err != nil {
				return err
			}

			var state any
			if spec.PostParse != nil {
				body, state, err = spec.PostParse(cmd, d, path, query, body)
				if err != nil {
					return err
				}
			}
			if spec.OfflineValidate != nil {
				if err := spec.OfflineValidate(cmd, d); err != nil {
					return err
				}
			}
			if d.Mutation || d.Download {
				dryRun, _ := cmd.Flags().GetBool("dry-run")
				if dryRun {
					fields, err := buildDryRunFields(spec, cmd, d, displayPath, query, body, state)
					if err != nil {
						return err
					}
					cli.PrintDryRun(Verb(d.Use), spec.ServiceName+" request", fields)
					return nil
				}
			}
			if spec.LiveValidate != nil {
				if err := spec.LiveValidate(cmd, d); err != nil {
					return err
				}
			}
			if d.Destructive {
				force, _ := cmd.Flags().GetBool("force")
				if !cli.Confirm(force, fmt.Sprintf("Proceed with %s %s?", d.Method, displayPath)) {
					return cli.ConfirmationError()
				}
			}
			if d.Download {
				return spec.RunDownload(cmd, d, path, query)
			}

			apiClient, err := spec.NewClient(cmd, d)
			if err != nil {
				return err
			}
			if len(secretPathValues) > 0 {
				// Fail closed if the client cannot mask secret paths.
				masker, ok := any(apiClient).(secretPathMasker)
				if !ok {
					return fmt.Errorf("internal %s contract error: %s declares secret path parameters but its client cannot mask them in debug output", spec.ServiceName, d.Use)
				}
				masker.MaskPathValues(secretPathValues)
			}
			response, err := spec.Execute(cmd, apiClient, d, path, query, body, state)
			if err != nil {
				return err
			}
			if spec.ResponseError != nil {
				if err := spec.ResponseError(d, response); err != nil {
					return err
				}
			}
			if response.Empty && !spec.AlwaysOutput {
				return nil
			}
			data := response.Data
			if spec.TransformOutput != nil {
				data, err = spec.TransformOutput(cmd, d, data)
				if err != nil {
					return err
				}
			}
			return cli.Output(cmd, data)
		},
	}
	registerFlags(cmd, spec, d)
	return cmd
}

func buildDryRunFields[C any](spec Spec[C], cmd *cobra.Command, d Descriptor, path string, query map[string]string, body, state any) (map[string]any, error) {
	if spec.DryRunFields != nil {
		return spec.DryRunFields(cmd, d, path, query, body, state)
	}
	return DefaultDryRunFields(d, path, query, body), nil
}

// DefaultDryRunFields returns a credential-redacted method, path, query, and body preview.
func DefaultDryRunFields(d Descriptor, path string, query map[string]string, body any) map[string]any {
	fields := map[string]any{"method": d.Method, "path": path}
	if len(query) > 0 {
		values := make(map[string]any, len(query))
		for key, value := range query {
			values[key] = value
		}
		fields["query"] = redact.JSON(values)
	}
	if body != nil {
		fields["body"] = redact.JSON(body)
	}
	return fields
}

// EmptyResponseError permits empty bodies only at declared success statuses.
func EmptyResponseError(serviceName string, d Descriptor, response client.HTTPResponse) error {
	if response.Empty && !d.HasEmptyStatus(response.StatusCode) {
		return fmt.Errorf("%s API returned an empty HTTP %d response for %s %s; expected JSON", serviceName, response.StatusCode, d.Method, d.Path)
	}
	return nil
}

func registerFlags[C any](cmd *cobra.Command, spec Spec[C], d Descriptor) {
	for _, parameter := range d.Paths {
		cmd.Flags().String(parameter.Flag, "", parameter.Usage)
		_ = cmd.MarkFlagRequired(parameter.Flag)
	}
	for _, parameter := range d.Queries {
		cmd.Flags().String(parameter.Flag, "", parameter.Usage)
		if parameter.Required {
			_ = cmd.MarkFlagRequired(parameter.Flag)
		}
	}
	if d.Body != nil {
		cmd.Flags().String("body", "", d.Body.Usage)
		if d.Body.Kind != OptionalObjectBody && d.Body.Kind != OptionalJSONBody {
			_ = cmd.MarkFlagRequired("body")
		}
	}
	if d.Mutation || d.Download {
		usage := spec.DryRunUsage
		if usage == "" {
			usage = "Preview the validated request without calling the API"
		}
		cmd.Flags().Bool("dry-run", false, usage)
	}
	// Register force once for deletion and download-overwrite confirmation.
	if d.Destructive || d.Download {
		usage := "Skip the destructive-action confirmation"
		if !d.Destructive {
			usage = "Skip confirmation before replacing an existing output file"
		}
		cmd.Flags().Bool("force", false, usage)
	}
	if d.Download {
		cmd.Flags().String("output-file", "", "Path for the downloaded certificate ZIP")
		_ = cmd.MarkFlagRequired("output-file")
	}
	if d.SecretResponse {
		usage := spec.ShowSecretUsage
		if usage == "" {
			usage = "Print returned credential material in command output"
		}
		cmd.Flags().Bool("show-secret", false, usage)
	}
	if spec.ExtraFlags != nil {
		spec.ExtraFlags(cmd, d)
	}
}
