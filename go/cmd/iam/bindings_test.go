package iam

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

func TestIAMBindingCommandsMatchDocumentedContracts(t *testing.T) {
	tests := []struct {
		name      string
		route     string
		method    string
		path      string
		configure func(*cobra.Command)
	}{
		{
			name: "attach IAM user to group", route: "group iam-user attach", method: http.MethodPost, path: "/v1/groups/group-1/iam-users/user-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("group-id", "group-1")
				_ = cmd.Flags().Set("iam-user-id", "user-1")
			},
		},
		{
			name: "detach IAM user from group", route: "group iam-user detach", method: http.MethodDelete, path: "/v1/groups/group-1/iam-users/user-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("group-id", "group-1")
				_ = cmd.Flags().Set("iam-user-id", "user-1")
			},
		},
		{
			name: "attach policy to group", route: "policy group attach", method: http.MethodPost, path: "/v1/policies/policy-1/groups/group-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("policy-id", "policy-1")
				_ = cmd.Flags().Set("group-id", "group-1")
			},
		},
		{
			name: "detach policy from group", route: "policy group detach", method: http.MethodDelete, path: "/v1/policies/policy-1/groups/group-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("policy-id", "policy-1")
				_ = cmd.Flags().Set("group-id", "group-1")
			},
		},
		{
			name: "attach policy to IAM user", route: "policy iam-user attach", method: http.MethodPost, path: "/v1/policies/policy-1/iam-users/user-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("policy-id", "policy-1")
				_ = cmd.Flags().Set("iam-user-id", "user-1")
			},
		},
		{
			name: "detach policy from IAM user", route: "policy iam-user detach", method: http.MethodDelete, path: "/v1/policies/policy-1/iam-users/user-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("policy-id", "policy-1")
				_ = cmd.Flags().Set("iam-user-id", "user-1")
			},
		},
		{
			name: "attach policy to service account", route: "policy service-account attach", method: http.MethodPost, path: "/v1/policies/policy-1/service-accounts/service-account-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("policy-id", "policy-1")
				_ = cmd.Flags().Set("service-account-id", "service-account-1")
			},
		},
		{
			name: "detach policy from service account", route: "policy service-account detach", method: http.MethodDelete, path: "/v1/policies/policy-1/service-accounts/service-account-1",
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("policy-id", "policy-1")
				_ = cmd.Flags().Set("service-account-id", "service-account-1")
			},
		},
	}

	installIAMTestClients(t,
		func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodGet || request.URL.Path != "/v1/auth/userinfo" {
				t.Errorf("identity request = %s %s, want GET /v1/auth/userinfo", request.Method, request.URL.Path)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"userId":"actor-1","userType":"iam-user"}`))
		},
		func(writer http.ResponseWriter, request *http.Request) {
			t.Fatalf("unexpected default policies request: %s %s", request.Method, request.URL.Path)
		},
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := fixtureForRequest(t, test.method, test.path)
			if response, ok := record.Responses["204"]; !ok || len(response.Content) != 0 {
				t.Fatal("binding fixture must require empty HTTP 204")
			}
			command := iamReadCommand(t, test.route)
			test.configure(command)
			_ = command.Flags().Set("force", "true")

			previousPolicies := policiesClientFactory
			policiesClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
				return mutationAssertingClient(t, test.method, test.path), nil
			}
			t.Cleanup(func() { policiesClientFactory = previousPolicies })

			if err := command.RunE(command, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestIAMBindingCommandsDryRunWithoutAPIOrConfirmation(t *testing.T) {
	accountsCalls := 0
	policiesCalls := 0
	previousAccounts := accountsClientFactory
	previousPolicies := policiesClientFactory
	accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		accountsCalls++
		return nil, fmt.Errorf("accounts client should not be created during dry run")
	}
	policiesClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		policiesCalls++
		return nil, fmt.Errorf("policies client should not be created during dry run")
	}
	t.Cleanup(func() {
		accountsClientFactory = previousAccounts
		policiesClientFactory = previousPolicies
	})

	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })

	for _, route := range []string{
		"group iam-user attach",
		"group iam-user detach",
		"policy group attach",
		"policy group detach",
		"policy iam-user attach",
		"policy iam-user detach",
		"policy service-account attach",
		"policy service-account detach",
	} {
		t.Run(route, func(t *testing.T) {
			command := iamReadCommand(t, route)
			for _, flag := range []string{"group-id", "policy-id", "iam-user-id", "service-account-id"} {
				if command.Flags().Lookup(flag) != nil {
					_ = command.Flags().Set(flag, "value-1")
				}
			}
			_ = command.Flags().Set("dry-run", "true")
			if err := command.RunE(command, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
	if accountsCalls != 0 || policiesCalls != 0 {
		t.Fatalf("dry run created %d Accounts and %d Policies clients", accountsCalls, policiesCalls)
	}
}

func TestIAMBindingCommandsRejectDirectSelfTarget(t *testing.T) {
	installIAMTestClients(t,
		func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"userId":"actor-1","userType":"iam-user"}`))
		},
		func(writer http.ResponseWriter, request *http.Request) {
			t.Fatalf("self-target protection made policies request: %s %s", request.Method, request.URL.Path)
		},
	)

	command := iamReadCommand(t, "policy iam-user attach")
	_ = command.Flags().Set("policy-id", "policy-1")
	_ = command.Flags().Set("iam-user-id", "actor-1")
	_ = command.Flags().Set("dry-run", "false")
	_ = command.Flags().Set("force", "true")
	if err := command.RunE(command, nil); err == nil || !strings.Contains(err.Error(), "current IAM identity") {
		t.Fatalf("self-target error = %v, want current identity rejection", err)
	}
}

func TestIAMBindingCommandsRejectServiceAccountSelfTarget(t *testing.T) {
	installIAMTestClients(t,
		func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"userId":"service-account-1","userType":"user-sa"}`))
		},
		func(writer http.ResponseWriter, request *http.Request) {
			t.Fatalf("self-target protection made policies request: %s %s", request.Method, request.URL.Path)
		},
	)

	command := iamReadCommand(t, "policy service-account attach")
	_ = command.Flags().Set("policy-id", "policy-1")
	_ = command.Flags().Set("service-account-id", "service-account-1")
	_ = command.Flags().Set("force", "true")
	if err := command.RunE(command, nil); err == nil || !strings.Contains(err.Error(), "current IAM identity") {
		t.Fatalf("service-account self-target error = %v, want current identity rejection", err)
	}
}

func TestIAMMutationEndpointOverrideIsRejected(t *testing.T) {
	command := &cobra.Command{Use: "test"}
	command.Flags().String("endpoint-url", "", "")
	if err := command.Flags().Set("endpoint-url", "https://override.greennode.ai"); err != nil {
		t.Fatal(err)
	}
	if err := rejectIAMMutationEndpointOverride(command); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("endpoint override error = %v, want rejection", err)
	}
}

func mutationAssertingClient(t *testing.T, method, path string) *client.GreennodeClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer offline-token" {
			t.Errorf("Authorization = %q, want offline bearer token", got)
		}
		if request.Method != method || request.URL.Path != path {
			t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, method, path)
		}
		if request.URL.RawQuery != "" {
			t.Errorf("query = %q, want none", request.URL.RawQuery)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) != 0 {
			t.Errorf("unexpected request body: %s", body)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	tokens := auth.NewMachineTokenProvider("fixture-client", "fixture-secret", "http://127.0.0.1:1/fixture-token")
	tokens.SetToken("offline-token", time.Now().Add(time.Hour))
	return client.NewGreennodeClient(server.URL, tokens, 0, time.Second, true, false)
}
