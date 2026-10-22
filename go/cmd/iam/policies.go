package iam

import "github.com/spf13/cobra"

var (
	actionCmd      = newCommandGroup("action", "Inspect IAM policy actions")
	listActionsCmd = newReadCommand(
		"list",
		"List available IAM actions",
		createPoliciesClient,
		staticPath("/v1/actions"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return optionalQuery(cmd, "product", "product")
		},
	)

	groupCmd      = newCommandGroup("group", "Manage IAM groups")
	listGroupsCmd = newReadCommand(
		"list",
		"List IAM groups",
		createPoliciesClient,
		staticPath("/v1/groups"),
		noQuery,
	)
	getGroupCmd = newReadCommand(
		"get",
		"Get an IAM group",
		createPoliciesClient,
		pathWithID("/v1/groups/", "group-id", ""),
		noQuery,
	)
	groupPolicyCmd       = newCommandGroup("policy", "Inspect policies attached to an IAM group")
	listGroupPoliciesCmd = newReadCommand(
		"list",
		"List policies attached to an IAM group",
		createPoliciesClient,
		pathWithID("/v1/groups/", "group-id", "/policies"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "name", "name")
		},
	)
	groupTagCmd      = newCommandGroup("tag", "Manage IAM-group tags")
	listGroupTagsCmd = newReadCommand(
		"list",
		"List tags for an IAM group",
		createPoliciesClient,
		pathWithID("/v1/groups/", "group-id", "/tags"),
		noQuery,
	)
	groupIAMUserCmd         = newCommandGroup("iam-user", "Manage IAM users in an IAM group")
	attachIAMUserToGroupCmd = newIAMBindingCommand(
		"attach",
		"Attach an IAM user to an IAM group",
		pathWithTwoIDs("/v1/groups/", "group-id", "/iam-users/", "iam-user-id", ""),
		"iam-user-id",
	)
	detachIAMUserFromGroupCmd = newIAMBindingCommand(
		"detach",
		"Detach an IAM user from an IAM group",
		pathWithTwoIDs("/v1/groups/", "group-id", "/iam-users/", "iam-user-id", ""),
		"iam-user-id",
	)

	policyCmd       = newCommandGroup("policy", "Manage IAM policies")
	listPoliciesCmd = newReadCommand(
		"list",
		"List IAM policies",
		createPoliciesClient,
		staticPath("/v1/policies"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "name", "name")
		},
	)
	getPolicyCmd = newReadCommand(
		"get",
		"Get an IAM policy",
		createPoliciesClient,
		pathWithID("/v1/policies/", "policy-id", ""),
		noQuery,
	)
	policyGroupCmd      = newCommandGroup("group", "Inspect groups attached to an IAM policy")
	listPolicyGroupsCmd = newReadCommand(
		"list",
		"List IAM groups attached to a policy",
		createPoliciesClient,
		pathWithID("/v1/policies/", "policy-id", "/groups"),
		noQuery,
	)
	attachPolicyToGroupCmd = newIAMBindingCommand(
		"attach",
		"Attach an IAM policy to an IAM group",
		pathWithTwoIDs("/v1/policies/", "policy-id", "/groups/", "group-id", ""),
		"",
	)
	detachPolicyFromGroupCmd = newIAMBindingCommand(
		"detach",
		"Detach an IAM policy from an IAM group",
		pathWithTwoIDs("/v1/policies/", "policy-id", "/groups/", "group-id", ""),
		"",
	)
	policyIAMUserCmd      = newCommandGroup("iam-user", "Inspect IAM users attached to an IAM policy")
	listPolicyIAMUsersCmd = newReadCommand(
		"list",
		"List IAM users attached to a policy",
		createPoliciesClient,
		pathWithID("/v1/policies/", "policy-id", "/iam-users"),
		noQuery,
	)
	attachPolicyToIAMUserCmd = newIAMBindingCommand(
		"attach",
		"Attach an IAM policy to an IAM user",
		pathWithTwoIDs("/v1/policies/", "policy-id", "/iam-users/", "iam-user-id", ""),
		"iam-user-id",
	)
	detachPolicyFromIAMUserCmd = newIAMBindingCommand(
		"detach",
		"Detach an IAM policy from an IAM user",
		pathWithTwoIDs("/v1/policies/", "policy-id", "/iam-users/", "iam-user-id", ""),
		"iam-user-id",
	)
	policyServiceAccountCmd      = newCommandGroup("service-account", "Inspect service accounts attached to an IAM policy")
	listPolicyServiceAccountsCmd = newReadCommand(
		"list",
		"List service accounts attached to a policy",
		createPoliciesClient,
		pathWithID("/v1/policies/", "policy-id", "/service-accounts"),
		noQuery,
	)
	attachPolicyToServiceAccountCmd = newIAMBindingCommand(
		"attach",
		"Attach an IAM policy to a service account",
		pathWithTwoIDs("/v1/policies/", "policy-id", "/service-accounts/", "service-account-id", ""),
		"service-account-id",
	)
	detachPolicyFromServiceAccountCmd = newIAMBindingCommand(
		"detach",
		"Detach an IAM policy from a service account",
		pathWithTwoIDs("/v1/policies/", "policy-id", "/service-accounts/", "service-account-id", ""),
		"service-account-id",
	)
	policyTagCmd      = newCommandGroup("tag", "Manage IAM-policy tags")
	listPolicyTagsCmd = newReadCommand(
		"list",
		"List tags for an IAM policy",
		createPoliciesClient,
		pathWithID("/v1/policies/", "policy-id", "/tags"),
		noQuery,
	)

	productCmd      = newCommandGroup("product", "Inspect IAM products")
	listProductsCmd = newReadCommand(
		"list",
		"List products available to IAM policies",
		createPoliciesClient,
		staticPath("/v1/products"),
		noQuery,
	)

	resourceCmd      = newCommandGroup("resource", "Inspect IAM policy resources")
	listResourcesCmd = newReadCommand(
		"list",
		"List IAM policy resources",
		createPoliciesClient,
		staticPath("/v1/resources"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return optionalQuery(cmd, "product", "product")
		},
	)

	iamUserGroupCmd      = newCommandGroup("group", "Inspect groups for an IAM user")
	listIAMUserGroupsCmd = newReadCommand(
		"list",
		"List groups for an IAM user",
		createPoliciesClient,
		pathWithID("/v1/user-attachments/iam-users/", "iam-user-id", "/groups"),
		noQuery,
	)
	iamUserPolicyCmd       = newCommandGroup("policy", "Inspect policies attached to an IAM user")
	listIAMUserPoliciesCmd = newReadCommand(
		"list",
		"List policies attached to an IAM user",
		createPoliciesClient,
		pathWithID("/v1/user-attachments/iam-users/", "iam-user-id", "/policies"),
		func(cmd *cobra.Command) (map[string]string, error) {
			return paginationQueryWithOptional(cmd, "name", "name")
		},
	)
)

func init() {
	actionCmd.AddCommand(listActionsCmd)

	groupCmd.AddCommand(
		listGroupsCmd,
		getGroupCmd,
		groupPolicyCmd,
		groupTagCmd,
		groupIAMUserCmd,
		mutationCommand("group-create"),
		mutationCommand("group-update"),
		mutationCommand("group-delete"),
	)
	groupPolicyCmd.AddCommand(listGroupPoliciesCmd)
	groupTagCmd.AddCommand(listGroupTagsCmd, mutationCommand("group-tag-create"))
	groupIAMUserCmd.AddCommand(attachIAMUserToGroupCmd, detachIAMUserFromGroupCmd)

	policyCmd.AddCommand(
		listPoliciesCmd,
		getPolicyCmd,
		policyGroupCmd,
		policyIAMUserCmd,
		policyServiceAccountCmd,
		policyTagCmd,
		mutationCommand("policy-compose"),
		mutationCommand("policy-create"),
		mutationCommand("policy-update"),
		mutationCommand("policy-delete"),
	)
	policyGroupCmd.AddCommand(listPolicyGroupsCmd, attachPolicyToGroupCmd, detachPolicyFromGroupCmd)
	policyIAMUserCmd.AddCommand(listPolicyIAMUsersCmd, attachPolicyToIAMUserCmd, detachPolicyFromIAMUserCmd)
	policyServiceAccountCmd.AddCommand(listPolicyServiceAccountsCmd, attachPolicyToServiceAccountCmd, detachPolicyFromServiceAccountCmd)
	policyTagCmd.AddCommand(listPolicyTagsCmd, mutationCommand("policy-tag-create"), mutationCommand("policy-tag-delete"))

	productCmd.AddCommand(listProductsCmd)
	resourceCmd.AddCommand(listResourcesCmd)

	IAMUserCmd.AddCommand(iamUserGroupCmd, iamUserPolicyCmd)
	iamUserGroupCmd.AddCommand(listIAMUserGroupsCmd)
	iamUserPolicyCmd.AddCommand(listIAMUserPoliciesCmd)

	listActionsCmd.Flags().String("product", "", "Filter actions by product")

	addRequiredIDFlag(getGroupCmd, "group-id", "IAM group ID")
	addRequiredIDFlag(listGroupPoliciesCmd, "group-id", "IAM group ID")
	addPaginationFlags(listGroupPoliciesCmd)
	listGroupPoliciesCmd.Flags().String("name", "", "Filter by policy name")
	addRequiredIDFlag(listGroupTagsCmd, "group-id", "IAM group ID")
	addIAMBindingFlags(attachIAMUserToGroupCmd,
		struct{ name, usage string }{"group-id", "IAM group ID"},
		struct{ name, usage string }{"iam-user-id", "IAM user ID"},
	)
	addIAMBindingFlags(detachIAMUserFromGroupCmd,
		struct{ name, usage string }{"group-id", "IAM group ID"},
		struct{ name, usage string }{"iam-user-id", "IAM user ID"},
	)

	addPaginationFlags(listPoliciesCmd)
	listPoliciesCmd.Flags().String("name", "", "Filter by policy name")
	addRequiredIDFlag(getPolicyCmd, "policy-id", "IAM policy ID")
	addRequiredIDFlag(listPolicyGroupsCmd, "policy-id", "IAM policy ID")
	addIAMBindingFlags(attachPolicyToGroupCmd,
		struct{ name, usage string }{"policy-id", "IAM policy ID"},
		struct{ name, usage string }{"group-id", "IAM group ID"},
	)
	addIAMBindingFlags(detachPolicyFromGroupCmd,
		struct{ name, usage string }{"policy-id", "IAM policy ID"},
		struct{ name, usage string }{"group-id", "IAM group ID"},
	)
	addRequiredIDFlag(listPolicyIAMUsersCmd, "policy-id", "IAM policy ID")
	addIAMBindingFlags(attachPolicyToIAMUserCmd,
		struct{ name, usage string }{"policy-id", "IAM policy ID"},
		struct{ name, usage string }{"iam-user-id", "IAM user ID"},
	)
	addIAMBindingFlags(detachPolicyFromIAMUserCmd,
		struct{ name, usage string }{"policy-id", "IAM policy ID"},
		struct{ name, usage string }{"iam-user-id", "IAM user ID"},
	)
	addRequiredIDFlag(listPolicyServiceAccountsCmd, "policy-id", "IAM policy ID")
	addIAMBindingFlags(attachPolicyToServiceAccountCmd,
		struct{ name, usage string }{"policy-id", "IAM policy ID"},
		struct{ name, usage string }{"service-account-id", "Service account ID"},
	)
	addIAMBindingFlags(detachPolicyFromServiceAccountCmd,
		struct{ name, usage string }{"policy-id", "IAM policy ID"},
		struct{ name, usage string }{"service-account-id", "Service account ID"},
	)
	addRequiredIDFlag(listPolicyTagsCmd, "policy-id", "IAM policy ID")

	listResourcesCmd.Flags().String("product", "", "Filter resources by product")

	addRequiredIDFlag(listIAMUserGroupsCmd, "iam-user-id", "IAM user ID")
	addRequiredIDFlag(listIAMUserPoliciesCmd, "iam-user-id", "IAM user ID")
	addPaginationFlags(listIAMUserPoliciesCmd)
	listIAMUserPoliciesCmd.Flags().String("name", "", "Filter by policy name")
}
