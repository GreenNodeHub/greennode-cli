package iam

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/iamclient"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

type clientFactory func(*cobra.Command) (*client.GreennodeClient, error)

type readPath func(*cobra.Command) (string, error)

type queryBuilder func(*cobra.Command) (map[string]string, error)

var (
	accountsClientFactory clientFactory = iamclient.BuildAccountsClient
	policiesClientFactory clientFactory = iamclient.BuildPoliciesClient
)

func createAccountsClient(cmd *cobra.Command) (*client.GreennodeClient, error) {
	return accountsClientFactory(cmd)
}

func createPoliciesClient(cmd *cobra.Command) (*client.GreennodeClient, error) {
	return policiesClientFactory(cmd)
}

func outputResult(cmd *cobra.Command, data any) error {
	return cli.Output(cmd, data)
}

func newCommandGroup(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}

func newReadCommand(use, short string, buildClient clientFactory, buildPath readPath, buildQuery queryBuilder) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := buildPath(cmd)
			if err != nil {
				return err
			}

			query, err := buildQuery(cmd)
			if err != nil {
				return err
			}

			apiClient, err := buildClient(cmd)
			if err != nil {
				return err
			}

			response, err := apiClient.RequestWithStatusNoRetrySensitive("GET", path, query, nil)
			if err != nil {
				return err
			}
			showSecret, _ := cmd.Flags().GetBool("show-secret")
			if response.StatusCode != 200 || response.Empty {
				return fmt.Errorf("IAM read requires HTTP 200 JSON")
			}
			result := response.Data
			if !showSecret {
				if strings.Contains(path, "/s3-keys") || strings.Contains(path, "/swift-users") {
					result = redactIAMSecretResponse(result)
				} else {
					result = cli.RedactJSON(result)
				}
			}
			return outputResult(cmd, result)
		},
	}
	cmd.Flags().Bool("show-secret", false, "Print returned credential fields")
	return cmd
}

func staticPath(path string) readPath {
	return func(*cobra.Command) (string, error) {
		return path, nil
	}
}

func pathWithID(prefix, flagName, suffix string) readPath {
	return func(cmd *cobra.Command) (string, error) {
		id, err := requiredID(cmd, flagName)
		if err != nil {
			return "", err
		}
		return prefix + id + suffix, nil
	}
}

func pathWithTwoIDs(prefix, firstFlag, middle, secondFlag, suffix string) readPath {
	return func(cmd *cobra.Command) (string, error) {
		first, err := requiredID(cmd, firstFlag)
		if err != nil {
			return "", err
		}
		second, err := requiredID(cmd, secondFlag)
		if err != nil {
			return "", err
		}
		return prefix + first + middle + second + suffix, nil
	}
}

func requiredID(cmd *cobra.Command, flagName string) (string, error) {
	value, _ := cmd.Flags().GetString(flagName)
	if err := validator.ValidateID(value, flagName); err != nil {
		return "", err
	}
	return value, nil
}

func noQuery(*cobra.Command) (map[string]string, error) {
	return nil, nil
}

func paginationQuery(cmd *cobra.Command) (map[string]string, error) {
	pageNumber, _ := cmd.Flags().GetInt("page-number")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	if pageNumber < 0 {
		return nil, fmt.Errorf("page-number must be greater than or equal to 0")
	}
	if pageSize <= 0 {
		return nil, fmt.Errorf("page-size must be greater than 0")
	}
	return map[string]string{
		"pageNumber": strconv.Itoa(pageNumber),
		"pageSize":   strconv.Itoa(pageSize),
	}, nil
}

func paginationQueryWithOptional(cmd *cobra.Command, flagName, parameterName string) (map[string]string, error) {
	query, err := paginationQuery(cmd)
	if err != nil {
		return nil, err
	}
	if value, _ := cmd.Flags().GetString(flagName); value != "" {
		query[parameterName] = value
	}
	return query, nil
}

func optionalQuery(cmd *cobra.Command, flagName, parameterName string) (map[string]string, error) {
	if value, _ := cmd.Flags().GetString(flagName); value != "" {
		return map[string]string{parameterName: value}, nil
	}
	return nil, nil
}

func addRequiredIDFlag(cmd *cobra.Command, name, usage string) {
	cmd.Flags().String(name, "", usage)
	_ = cmd.MarkFlagRequired(name)
}

func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page-number", 0, "Page number (0-based)")
	cmd.Flags().Int("page-size", 10, "Number of items per page")
}
