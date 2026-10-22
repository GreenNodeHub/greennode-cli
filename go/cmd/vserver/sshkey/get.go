package sshkey

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an SSH key",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("sshkey-id", "", "SSH key ID (required)")
	getCmd.Flags().Bool("show-secret", false, "Print returned private key material in command output")
	if err := getCmd.MarkFlagRequired("sshkey-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "sshkey-id", err))
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	sshKeyID, _ := cmd.Flags().GetString("sshkey-id")
	if err := validator.ValidateID(sshKeyID, "sshkey-id"); err != nil {
		return err
	}

	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := requestKey(apiClient, "GET", fmt.Sprintf("/v2/%s/sshKeys/%s", projectID, sshKeyID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get SSH key %s: %w", sshKeyID, err)
	}
	return outputKeyList(cmd, cfg, result)
}
