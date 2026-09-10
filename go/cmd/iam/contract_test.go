package iam

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func installIAMTestClients(t *testing.T, accountsHandler, policiesHandler http.HandlerFunc) {
	t.Helper()
	accountsServer := httptest.NewServer(accountsHandler)
	policiesServer := httptest.NewServer(policiesHandler)
	t.Cleanup(accountsServer.Close)
	t.Cleanup(policiesServer.Close)

	newTestClient := func(baseURL string) *client.GreennodeClient {
		tokens := auth.NewMachineTokenProvider("fixture-client", "fixture-secret", "http://127.0.0.1:1/fixture-token")
		tokens.SetToken("offline-token", time.Now().Add(time.Hour))
		return client.NewGreennodeClient(baseURL, tokens, 0, time.Second, true, false)
	}

	previousAccounts := accountsClientFactory
	previousPolicies := policiesClientFactory
	accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		return newTestClient(accountsServer.URL), nil
	}
	policiesClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		return newTestClient(policiesServer.URL), nil
	}
	t.Cleanup(func() {
		accountsClientFactory = previousAccounts
		policiesClientFactory = previousPolicies
	})
}

func iamReadCommand(t *testing.T, route string) *cobra.Command {
	t.Helper()
	command, _, err := IamCmd.Find(strings.Fields(route))
	if err != nil || command == nil || command.RunE == nil {
		t.Fatalf("IAM route %q is not runnable: %v", route, err)
	}
	resetIAMCommandFlags(t, command)
	t.Cleanup(func() { resetIAMCommandFlags(t, command) })
	return command
}

func resetIAMCommandFlags(t *testing.T, command *cobra.Command) {
	t.Helper()
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset %s flag: %v", flag.Name, err)
		}
		flag.Changed = false
	})
}

func assertIAMRequest(t *testing.T, request *http.Request, path string, query url.Values) {
	t.Helper()
	if got := request.Header.Get("Authorization"); got != "Bearer offline-token" {
		t.Errorf("Authorization = %q, want offline bearer token", got)
	}
	if request.Method != http.MethodGet || request.URL.Path != path {
		t.Errorf("request = %s %s, want GET %s", request.Method, request.URL.Path, path)
	}
	if got := request.URL.Query(); !reflect.DeepEqual(got, query) {
		t.Errorf("query = %#v, want %#v", got, query)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Errorf("unexpected request body: %s", body)
	}
}

func writeIAMResponse(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(`{"data":[]}`))
}

func TestIAMReadRoutesMatchDocumentedContracts(t *testing.T) {
	tests := []struct {
		name      string
		route     string
		api       string
		configure func(*cobra.Command)
		path      string
		query     url.Values
	}{
		{
			name: "whoami", route: "whoami", api: "accounts", path: "/v1/auth/userinfo", query: url.Values{},
			configure: func(*cobra.Command) {},
		},
		{
			name: "list identity providers", route: "identity-provider list", api: "accounts", path: "/v1/identity-providers",
			query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}}, configure: func(*cobra.Command) {},
		},
		{
			name: "get identity provider", route: "identity-provider get", api: "accounts", path: "/v1/identity-providers/idp-1", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("identity-provider-id", "idp-1") },
		},
		{
			name: "list permission mappers", route: "identity-provider permission-mapper list", api: "accounts", path: "/v1/identity-providers/idp-1/permission-mappers", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("identity-provider-id", "idp-1") },
		},
		{
			name: "get permission mapper", route: "identity-provider permission-mapper get", api: "accounts", path: "/v1/identity-providers/idp-1/permission-mappers/mapper-1", query: url.Values{},
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("identity-provider-id", "idp-1")
				_ = cmd.Flags().Set("mapper-id", "mapper-1")
			},
		},
		{
			name: "list s3 keys", route: "s3-key list", api: "accounts", path: "/v1/s3-keys",
			query:     url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "searchByNameOrAccessKey": {"deploy"}},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("search", "deploy") },
		},
		{
			name: "get s3 key", route: "s3-key get", api: "accounts", path: "/v1/s3-keys/key-1", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("s3-key-id", "key-1") },
		},
		{
			name: "list service accounts", route: "service-account list", api: "accounts", path: "/v1/service-accounts", query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}},
			configure: func(*cobra.Command) {},
		},
		{
			name: "get service account", route: "service-account get", api: "accounts", path: "/v1/service-accounts/sa-1", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("service-account-id", "sa-1") },
		},
		{
			name: "list service account s3 keys", route: "service-account s3-key list", api: "accounts", path: "/v1/service-accounts/sa-1/s3-keys",
			query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "searchByNameOrAccessKey": {"deploy"}},
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("service-account-id", "sa-1")
				_ = cmd.Flags().Set("search", "deploy")
			},
		},
		{
			name: "list service account swift users", route: "service-account swift-user list", api: "accounts", path: "/v1/service-accounts/sa-1/swift-users",
			query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "searchByUsername": {"backup"}},
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("service-account-id", "sa-1")
				_ = cmd.Flags().Set("search", "backup")
			},
		},
		{
			name: "list service account tags", route: "service-account tag list", api: "accounts", path: "/v1/service-accounts/sa-1/tags", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("service-account-id", "sa-1") },
		},
		{
			name: "list service account trusted roots", route: "service-account trusted-root list", api: "accounts", path: "/v1/service-accounts/sa-1/trusted-roots", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("service-account-id", "sa-1") },
		},
		{
			name: "list service account policies", route: "service-account policy list", api: "policies", path: "/v1/user-attachments/service-accounts/sa-1/policies",
			query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "name": {"reader"}},
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("service-account-id", "sa-1")
				_ = cmd.Flags().Set("name", "reader")
			},
		},
		{
			name: "list swift users", route: "swift-user list", api: "accounts", path: "/v1/swift-users",
			query:     url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "searchByUsername": {"backup"}},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("search", "backup") },
		},
		{
			name: "get swift user", route: "swift-user get", api: "accounts", path: "/v1/swift-users/swift-1", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("swift-user-id", "swift-1") },
		},
		{
			name: "list iam users", route: "iam-user list", api: "accounts", path: "/v1/iam-users",
			query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}}, configure: func(*cobra.Command) {},
		},
		{
			name: "get iam user", route: "iam-user get", api: "accounts", path: "/v1/iam-users/user-1", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("iam-user-id", "user-1") },
		},
		{
			name: "list iam user tags", route: "iam-user tag list", api: "accounts", path: "/v1/iam-users/user-1/tags", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("iam-user-id", "user-1") },
		},
		{
			name: "list actions", route: "action list", api: "policies", path: "/v1/actions", query: url.Values{"product": {"vks"}},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("product", "vks") },
		},
		{
			name: "list groups", route: "group list", api: "policies", path: "/v1/groups", query: url.Values{},
			configure: func(*cobra.Command) {},
		},
		{
			name: "get group", route: "group get", api: "policies", path: "/v1/groups/group-1", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("group-id", "group-1") },
		},
		{
			name: "list group policies", route: "group policy list", api: "policies", path: "/v1/groups/group-1/policies",
			query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "name": {"reader"}},
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("group-id", "group-1")
				_ = cmd.Flags().Set("name", "reader")
			},
		},
		{
			name: "list group tags", route: "group tag list", api: "policies", path: "/v1/groups/group-1/tags", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("group-id", "group-1") },
		},
		{
			name: "list policies", route: "policy list", api: "policies", path: "/v1/policies",
			query:     url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "name": {"reader"}},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("name", "reader") },
		},
		{
			name: "get policy", route: "policy get", api: "policies", path: "/v1/policies/policy-1", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("policy-id", "policy-1") },
		},
		{
			name: "list policy groups", route: "policy group list", api: "policies", path: "/v1/policies/policy-1/groups", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("policy-id", "policy-1") },
		},
		{
			name: "list policy iam users", route: "policy iam-user list", api: "policies", path: "/v1/policies/policy-1/iam-users", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("policy-id", "policy-1") },
		},
		{
			name: "list policy service accounts", route: "policy service-account list", api: "policies", path: "/v1/policies/policy-1/service-accounts", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("policy-id", "policy-1") },
		},
		{
			name: "list policy tags", route: "policy tag list", api: "policies", path: "/v1/policies/policy-1/tags", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("policy-id", "policy-1") },
		},
		{
			name: "list products", route: "product list", api: "policies", path: "/v1/products", query: url.Values{},
			configure: func(*cobra.Command) {},
		},
		{
			name: "list resources", route: "resource list", api: "policies", path: "/v1/resources", query: url.Values{"product": {"vks"}},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("product", "vks") },
		},
		{
			name: "list iam user groups", route: "iam-user group list", api: "policies", path: "/v1/user-attachments/iam-users/user-1/groups", query: url.Values{},
			configure: func(cmd *cobra.Command) { _ = cmd.Flags().Set("iam-user-id", "user-1") },
		},
		{
			name: "list iam user policies", route: "iam-user policy list", api: "policies", path: "/v1/user-attachments/iam-users/user-1/policies",
			query: url.Values{"pageNumber": {"0"}, "pageSize": {"10"}, "name": {"reader"}},
			configure: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("iam-user-id", "user-1")
				_ = cmd.Flags().Set("name", "reader")
			},
		},
	}
	wantRoutes := make(map[string]bool, len(tests))
	for _, test := range tests {
		if wantRoutes[test.route] {
			t.Fatalf("duplicate IAM read test route %q", test.route)
		}
		wantRoutes[test.route] = true
	}
	gotRoutes := map[string]bool{}
	collectIAMReadRoutes(IamCmd, nil, gotRoutes)
	if !reflect.DeepEqual(gotRoutes, wantRoutes) {
		t.Fatalf("IAM read test routes = %v, want all registered read routes %v", sortedIAMRoutes(wantRoutes), sortedIAMRoutes(gotRoutes))
	}

	installIAMTestClients(t,
		func(writer http.ResponseWriter, request *http.Request) {
			writeIAMResponse(writer)
		},
		func(writer http.ResponseWriter, request *http.Request) {
			writeIAMResponse(writer)
		},
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := fixtureForRequest(t, http.MethodGet, test.path)
			assertPublishedQuery(t, record, test.query)
			command := iamReadCommand(t, test.route)
			test.configure(command)

			previousAccounts := accountsClientFactory
			previousPolicies := policiesClientFactory
			if test.api == "accounts" {
				accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
					return requestAssertingClient(t, test.path, test.query), nil
				}
			} else {
				policiesClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
					return requestAssertingClient(t, test.path, test.query), nil
				}
			}
			t.Cleanup(func() {
				accountsClientFactory = previousAccounts
				policiesClientFactory = previousPolicies
			})

			if err := command.RunE(command, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func collectIAMReadRoutes(command *cobra.Command, parents []string, routes map[string]bool) {
	path := append(parents, command.Name())
	children := command.Commands()
	if len(children) == 0 {
		if len(path) > 1 && command.Flags().Lookup("dry-run") == nil {
			routes[strings.Join(path[1:], " ")] = true
		}
		return
	}
	for _, child := range children {
		collectIAMReadRoutes(child, path, routes)
	}
}

func sortedIAMRoutes(routes map[string]bool) []string {
	result := make([]string, 0, len(routes))
	for route := range routes {
		result = append(result, route)
	}
	sort.Strings(result)
	return result
}

func requestAssertingClient(t *testing.T, path string, query url.Values) *client.GreennodeClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertIAMRequest(t, request, path, query)
		writeIAMResponse(writer)
	}))
	t.Cleanup(server.Close)
	tokens := auth.NewMachineTokenProvider("fixture-client", "fixture-secret", "http://127.0.0.1:1/fixture-token")
	tokens.SetToken("offline-token", time.Now().Add(time.Hour))
	return client.NewGreennodeClient(server.URL, tokens, 0, time.Second, true, false)
}

func TestIAMReadCommandsRejectUnsafeIdentifiersAndPagination(t *testing.T) {
	apiCalls := 0
	installIAMTestClients(t,
		func(writer http.ResponseWriter, request *http.Request) {
			apiCalls++
			writeIAMResponse(writer)
		},
		func(writer http.ResponseWriter, request *http.Request) {
			apiCalls++
			writeIAMResponse(writer)
		},
	)

	getCommand := iamReadCommand(t, "service-account get")
	_ = getCommand.Flags().Set("service-account-id", "unsafe/path")
	if err := getCommand.RunE(getCommand, nil); err == nil || !strings.Contains(err.Error(), "invalid service-account-id") {
		t.Fatalf("unsafe ID error = %v", err)
	}

	listCommand := iamReadCommand(t, "iam-user list")
	_ = listCommand.Flags().Set("page-number", "-1")
	if err := listCommand.RunE(listCommand, nil); err == nil || !strings.Contains(err.Error(), "page-number") {
		t.Fatalf("negative page error = %v", err)
	}

	if apiCalls != 0 {
		t.Fatalf("invalid input made %d API calls", apiCalls)
	}
}
