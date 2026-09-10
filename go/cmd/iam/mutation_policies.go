package iam

import "net/http"

func policiesMutationOperations() []iamMutationOperation {
	groupID := mutationPath("groupId", "group-id", "IAM group ID")
	policyID := mutationPath("id", "policy-id", "IAM policy ID")
	policyTagID := mutationPath("policyId", "policy-id", "IAM policy ID")
	return []iamMutationOperation{
		{
			Key: "group-create", Use: "create", Short: "Create an IAM group", Method: http.MethodPost, Path: "/v1/groups", BuildClient: createPoliciesClient,
			Body: &iamMutationBody{RequiredFields: []string{"name"}}, Status: http.StatusCreated, ResponseBody: true,
		},
		{
			Key: "group-update", Use: "update", Short: "Update an IAM group", Method: http.MethodPatch, Path: "/v1/groups/{groupId}", BuildClient: createPoliciesClient,
			Paths: []iamMutationPath{groupID}, Body: &iamMutationBody{RequiredFields: []string{"name"}}, Status: http.StatusNoContent,
		},
		{
			Key: "group-delete", Use: "delete", Short: "Delete an IAM group", Method: http.MethodDelete, Path: "/v1/groups/{groupId}", BuildClient: createPoliciesClient,
			Paths: []iamMutationPath{groupID}, Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "group-tag-create", Use: "create", Short: "Create tags for an IAM group", Method: http.MethodPost, Path: "/v1/groups/{groupId}/tags", BuildClient: createPoliciesClient,
			Paths: []iamMutationPath{groupID}, Body: &iamMutationBody{RequiredFields: []string{"tags"}}, Status: http.StatusCreated,
		},
		{
			Key: "policy-compose", Use: "compose", Short: "Compose IAM policy statements", Method: http.MethodPost, Path: "/v1/compose-policy", BuildClient: createPoliciesClient,
			Body: &iamMutationBody{Shape: iamMutationBodyArray}, Status: http.StatusOK, ResponseBody: true,
		},
		{
			Key: "policy-create", Use: "create", Short: "Create an IAM policy", Method: http.MethodPost, Path: "/v1/policies", BuildClient: createPoliciesClient,
			Body: &iamMutationBody{}, Status: http.StatusCreated, ResponseBody: true,
		},
		{
			Key: "policy-update", Use: "update", Short: "Update an IAM policy", Method: http.MethodPut, Path: "/v1/policies/{id}", BuildClient: createPoliciesClient,
			Paths: []iamMutationPath{policyID}, Body: &iamMutationBody{}, Status: http.StatusNoContent,
		},
		{
			Key: "policy-delete", Use: "delete", Short: "Delete an IAM policy", Method: http.MethodDelete, Path: "/v1/policies/{id}", BuildClient: createPoliciesClient,
			Paths: []iamMutationPath{policyID}, Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "policy-tag-create", Use: "create", Short: "Create tags for an IAM policy", Method: http.MethodPost, Path: "/v1/policies/{policyId}/tags", BuildClient: createPoliciesClient,
			Paths: []iamMutationPath{policyTagID}, Body: &iamMutationBody{}, Status: http.StatusCreated,
		},
		{
			Key: "policy-tag-delete", Use: "delete", Short: "Delete tags from an IAM policy", Method: http.MethodDelete, Path: "/v1/policies/{policyId}/tags", BuildClient: createPoliciesClient,
			Paths: []iamMutationPath{policyTagID}, Body: &iamMutationBody{}, Status: http.StatusNoContent, Destructive: true,
		},
	}
}

func allIAMMutationOperations() []iamMutationOperation {
	operations := accountsMutationOperations()
	return append(operations, policiesMutationOperations()...)
}
