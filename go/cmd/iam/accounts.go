package iam

import "github.com/spf13/cobra"

var (
	whoamiCmd = newReadCommand(
		"whoami",
		"Show the IAM identity for the current credentials",
		createAccountsClient,
		staticPath("/v1/auth/userinfo"),
		noQuery,
	)

	identityProviderCmd      = newCommandGroup("identity-provider", "Manage IAM identity providers")
	listIdentityProvidersCmd = newReadCommand(
		"list",
		"List identity providers",
		createAccountsClient,
		staticPath("/v1/identity-providers"),
		paginationQuery,
	)
	getIdentityProviderCmd = newReadCommand(
		"get",
		"Get an identity provider",
		createAccountsClient,
		pathWithID("/v1/identity-providers/", "identity-provider-id", ""),
		noQuery,
	)
	permissionMapperCmd      = newCommandGroup("permission-mapper", "Manage identity-provider permission mappers")
	listPermissionMappersCmd = newReadCommand(
		"list",
		"List permission mappers for an identity provider",
		createAccountsClient,
		pathWithID("/v1/identity-providers/", "identity-provider-id", "/permission-mappers"),
		noQuery,
	)
	getPermissionMapperCmd = newReadCommand(
		"get",
		"Get an identity-provider permission mapper",
		createAccountsClient,
		pathWithTwoIDs("/v1/identity-providers/", "identity-provider-id", "/permission-mappers/", "mapper-id", ""),
		noQuery,
	)

	s3KeyCmd      = newCommandGroup("s3-key", "Manage S3 keys")
	listS3KeysCmd = newReadCommand(
		"list",
		"List S3 keys",
		createAccountsClient,
		staticPath("/v1/s3-keys"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "search", "searchByNameOrAccessKey")
		},
	)
	getS3KeyCmd = newReadCommand(
		"get",
		"Get an S3 key",
		createAccountsClient,
		pathWithID("/v1/s3-keys/", "s3-key-id", ""),
		noQuery,
	)

	serviceAccountCmd      = newCommandGroup("service-account", "Manage IAM service accounts")
	listServiceAccountsCmd = newReadCommand(
		"list",
		"List service accounts",
		createAccountsClient,
		staticPath("/v1/service-accounts"),
		paginationQuery,
	)
	getServiceAccountCmd = newReadCommand(
		"get",
		"Get a service account",
		createAccountsClient,
		pathWithID("/v1/service-accounts/", "service-account-id", ""),
		noQuery,
	)
	serviceAccountS3KeyCmd      = newCommandGroup("s3-key", "Manage S3 keys attached to a service account")
	listServiceAccountS3KeysCmd = newReadCommand(
		"list",
		"List S3 keys attached to a service account",
		createAccountsClient,
		pathWithID("/v1/service-accounts/", "service-account-id", "/s3-keys"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "search", "searchByNameOrAccessKey")
		},
	)
	serviceAccountSwiftUserCmd      = newCommandGroup("swift-user", "Manage Swift users attached to a service account")
	listServiceAccountSwiftUsersCmd = newReadCommand(
		"list",
		"List Swift users attached to a service account",
		createAccountsClient,
		pathWithID("/v1/service-accounts/", "service-account-id", "/swift-users"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "search", "searchByUsername")
		},
	)
	serviceAccountTagCmd      = newCommandGroup("tag", "Manage service-account tags")
	listServiceAccountTagsCmd = newReadCommand(
		"list",
		"List tags for a service account",
		createAccountsClient,
		pathWithID("/v1/service-accounts/", "service-account-id", "/tags"),
		noQuery,
	)
	trustedRootCmd      = newCommandGroup("trusted-root", "Manage service-account trusted roots")
	listTrustedRootsCmd = newReadCommand(
		"list",
		"List trusted roots for a service account",
		createAccountsClient,
		pathWithID("/v1/service-accounts/", "service-account-id", "/trusted-roots"),
		noQuery,
	)
	serviceAccountPolicyCmd       = newCommandGroup("policy", "Inspect policies attached to a service account")
	listServiceAccountPoliciesCmd = newReadCommand(
		"list",
		"List policies attached to a service account",
		createPoliciesClient,
		pathWithID("/v1/user-attachments/service-accounts/", "service-account-id", "/policies"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "name", "name")
		},
	)

	swiftUserCmd      = newCommandGroup("swift-user", "Manage Swift users")
	listSwiftUsersCmd = newReadCommand(
		"list",
		"List Swift users",
		createAccountsClient,
		staticPath("/v1/swift-users"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "search", "searchByUsername")
		},
	)
	getSwiftUserCmd = newReadCommand(
		"get",
		"Get a Swift user",
		createAccountsClient,
		pathWithID("/v1/swift-users/", "swift-user-id", ""),
		noQuery,
	)

	IAMUserCmd        = newCommandGroup("iam-user", "Manage IAM users")
	iamUserCurrentCmd = newCommandGroup("current", "Manage the current IAM user")
	listIAMUsersCmd   = newReadCommand(
		"list",
		"List IAM users",
		createAccountsClient,
		staticPath("/v1/iam-users"),
		paginationQuery,
	)
	getIAMUserCmd = newReadCommand(
		"get",
		"Get an IAM user",
		createAccountsClient,
		pathWithID("/v1/iam-users/", "iam-user-id", ""),
		noQuery,
	)
	iamUserTagCmd      = newCommandGroup("tag", "Inspect IAM-user tags")
	listIAMUserTagsCmd = newReadCommand(
		"list",
		"List tags for an IAM user",
		createAccountsClient,
		pathWithID("/v1/iam-users/", "iam-user-id", "/tags"),
		noQuery,
	)
)

func init() {
	identityProviderCmd.AddCommand(listIdentityProvidersCmd, getIdentityProviderCmd, permissionMapperCmd, mutationCommand("identity-provider-create"), mutationCommand("identity-provider-update"), mutationCommand("identity-provider-delete"))
	permissionMapperCmd.AddCommand(listPermissionMappersCmd, getPermissionMapperCmd, mutationCommand("permission-mapper-create"), mutationCommand("permission-mapper-update"), mutationCommand("permission-mapper-delete"))

	s3KeyCmd.AddCommand(listS3KeysCmd, getS3KeyCmd, mutationCommand("s3-key-create"), mutationCommand("s3-key-update"), mutationCommand("s3-key-delete"))

	serviceAccountCmd.AddCommand(
		listServiceAccountsCmd,
		getServiceAccountCmd,
		serviceAccountS3KeyCmd,
		serviceAccountSwiftUserCmd,
		serviceAccountTagCmd,
		trustedRootCmd,
		serviceAccountPolicyCmd,
		mutationCommand("service-account-create"),
		mutationCommand("service-account-update"),
		mutationCommand("service-account-delete"),
		mutationCommand("service-account-activate"),
		mutationCommand("service-account-deactivate"),
		mutationCommand("service-account-reset-secret"),
	)
	serviceAccountS3KeyCmd.AddCommand(listServiceAccountS3KeysCmd, mutationCommand("service-account-s3-key-attach"), mutationCommand("service-account-s3-key-detach"))
	serviceAccountSwiftUserCmd.AddCommand(listServiceAccountSwiftUsersCmd, mutationCommand("service-account-swift-user-attach"), mutationCommand("service-account-swift-user-detach"))
	serviceAccountTagCmd.AddCommand(listServiceAccountTagsCmd, mutationCommand("service-account-tag-create"), mutationCommand("service-account-tag-delete"))
	trustedRootCmd.AddCommand(listTrustedRootsCmd, mutationCommand("service-account-trusted-root-attach"), mutationCommand("service-account-trusted-root-detach"))
	serviceAccountPolicyCmd.AddCommand(listServiceAccountPoliciesCmd)

	swiftUserCmd.AddCommand(listSwiftUsersCmd, getSwiftUserCmd, mutationCommand("swift-user-create"), mutationCommand("swift-user-update"), mutationCommand("swift-user-delete"))
	IAMUserCmd.AddCommand(
		listIAMUsersCmd,
		getIAMUserCmd,
		iamUserTagCmd,
		iamUserCurrentCmd,
		mutationCommand("iam-user-create"),
		mutationCommand("iam-user-delete"),
		mutationCommand("iam-user-activate"),
		mutationCommand("iam-user-activate-email"),
		mutationCommand("iam-user-activate-google"),
		mutationCommand("iam-user-deactivate"),
		mutationCommand("iam-user-deactivate-email"),
		mutationCommand("iam-user-deactivate-google"),
		mutationCommand("iam-user-reset-password"),
		mutationCommand("iam-user-send-verification-email"),
		mutationCommand("iam-user-set-up-email"),
		mutationCommand("iam-user-set-up-google"),
		mutationCommand("iam-user-verify-email"),
		mutationCommand("iam-user-verify-google"),
	)
	iamUserTagCmd.AddCommand(listIAMUserTagsCmd, mutationCommand("iam-user-tag-create"), mutationCommand("iam-user-tag-delete"))
	iamUserCurrentCmd.AddCommand(
		mutationCommand("iam-user-current-activate-email"),
		mutationCommand("iam-user-current-activate-google"),
		mutationCommand("iam-user-current-deactivate-email"),
		mutationCommand("iam-user-current-deactivate-google"),
		mutationCommand("iam-user-current-reset-password"),
		mutationCommand("iam-user-current-send-verification-email"),
		mutationCommand("iam-user-current-set-up-email"),
		mutationCommand("iam-user-current-set-up-google"),
		mutationCommand("iam-user-current-verify-email"),
	)

	addPaginationFlags(listIdentityProvidersCmd)
	addRequiredIDFlag(getIdentityProviderCmd, "identity-provider-id", "Identity provider ID")
	addRequiredIDFlag(listPermissionMappersCmd, "identity-provider-id", "Identity provider ID")
	addRequiredIDFlag(getPermissionMapperCmd, "identity-provider-id", "Identity provider ID")
	addRequiredIDFlag(getPermissionMapperCmd, "mapper-id", "Permission mapper ID")

	addPaginationFlags(listS3KeysCmd)
	listS3KeysCmd.Flags().String("search", "", "Filter by S3 key name or access key")
	addRequiredIDFlag(getS3KeyCmd, "s3-key-id", "S3 key ID")

	addRequiredIDFlag(getServiceAccountCmd, "service-account-id", "Service account ID")
	addPaginationFlags(listServiceAccountsCmd)
	addRequiredIDFlag(listServiceAccountS3KeysCmd, "service-account-id", "Service account ID")
	addPaginationFlags(listServiceAccountS3KeysCmd)
	listServiceAccountS3KeysCmd.Flags().String("search", "", "Filter by S3 key name or access key")
	addRequiredIDFlag(listServiceAccountSwiftUsersCmd, "service-account-id", "Service account ID")
	addPaginationFlags(listServiceAccountSwiftUsersCmd)
	listServiceAccountSwiftUsersCmd.Flags().String("search", "", "Filter by Swift username")
	addRequiredIDFlag(listServiceAccountTagsCmd, "service-account-id", "Service account ID")
	addRequiredIDFlag(listTrustedRootsCmd, "service-account-id", "Service account ID")
	addRequiredIDFlag(listServiceAccountPoliciesCmd, "service-account-id", "Service account ID")
	addPaginationFlags(listServiceAccountPoliciesCmd)
	listServiceAccountPoliciesCmd.Flags().String("name", "", "Filter by policy name")

	addPaginationFlags(listSwiftUsersCmd)
	listSwiftUsersCmd.Flags().String("search", "", "Filter by Swift username")
	addRequiredIDFlag(getSwiftUserCmd, "swift-user-id", "Swift user ID")

	addPaginationFlags(listIAMUsersCmd)
	addRequiredIDFlag(getIAMUserCmd, "iam-user-id", "IAM user ID")
	addRequiredIDFlag(listIAMUserTagsCmd, "iam-user-id", "IAM user ID")
}
