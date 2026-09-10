package iam

import "net/http"

func accountsMutationOperations() []iamMutationOperation {
	serviceAccountID := mutationPath("id", "service-account-id", "Service account ID")
	iamUserID := mutationPath("id", "iam-user-id", "IAM user ID")
	return []iamMutationOperation{
		{
			Key: "identity-provider-create", Use: "create", Short: "Create an identity provider", Method: http.MethodPost, Path: "/v1/identity-providers", BuildClient: createAccountsClient,
			Body: &iamMutationBody{RequiredFields: []string{"name", "type", "vendor", "ssoUrl"}}, Status: http.StatusOK, ResponseBody: true,
		},
		{
			Key: "identity-provider-update", Use: "update", Short: "Update an identity provider", Method: http.MethodPatch, Path: "/v1/identity-providers/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("id", "identity-provider-id", "Identity provider ID")}, Body: &iamMutationBody{RequiredFields: []string{"name"}}, Status: http.StatusNoContent,
		},
		{
			Key: "identity-provider-delete", Use: "delete", Short: "Delete an identity provider", Method: http.MethodDelete, Path: "/v1/identity-providers/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("id", "identity-provider-id", "Identity provider ID")}, Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "permission-mapper-create", Use: "create", Short: "Create an identity-provider permission mapper", Method: http.MethodPost, Path: "/v1/identity-providers/{idpId}/permission-mappers", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("idpId", "identity-provider-id", "Identity provider ID")}, Body: &iamMutationBody{RequiredFields: []string{"name", "claimName", "claimValues", "groups"}}, Status: http.StatusCreated, ResponseBody: true,
		},
		{
			Key: "permission-mapper-update", Use: "update", Short: "Update an identity-provider permission mapper", Method: http.MethodPatch, Path: "/v1/identity-providers/{idpId}/permission-mappers/{mapperId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("idpId", "identity-provider-id", "Identity provider ID"), mutationPath("mapperId", "mapper-id", "Permission mapper ID")}, Body: &iamMutationBody{}, Status: http.StatusNoContent,
		},
		{
			Key: "permission-mapper-delete", Use: "delete", Short: "Delete an identity-provider permission mapper", Method: http.MethodDelete, Path: "/v1/identity-providers/{idpId}/permission-mappers/{mapperId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("idpId", "identity-provider-id", "Identity provider ID"), mutationPath("mapperId", "mapper-id", "Permission mapper ID")}, Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "s3-key-create", Use: "create", Short: "Create an S3 key", Method: http.MethodPost, Path: "/v1/s3-keys", BuildClient: createAccountsClient,
			Body: &iamMutationBody{RequiredFields: []string{"regionId", "projectId"}}, Status: http.StatusCreated, ResponseBody: true, SecretResponse: true,
		},
		{
			Key: "s3-key-update", Use: "update", Short: "Update an S3 key", Method: http.MethodPatch, Path: "/v1/s3-keys/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("id", "s3-key-id", "S3 key ID")}, Body: &iamMutationBody{}, Status: http.StatusNoContent,
		},
		{
			Key: "s3-key-delete", Use: "delete", Short: "Delete an S3 key", Method: http.MethodDelete, Path: "/v1/s3-keys/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("id", "s3-key-id", "S3 key ID")}, Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "service-account-create", Use: "create", Short: "Create a service account", Method: http.MethodPost, Path: "/v1/service-accounts", BuildClient: createAccountsClient,
			Body: &iamMutationBody{RequiredFields: []string{"name"}}, Status: http.StatusCreated, AllowUnexpectedBody: true,
		},
		{
			Key: "service-account-update", Use: "update", Short: "Update a service account", Method: http.MethodPatch, Path: "/v1/service-accounts/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID}, Body: &iamMutationBody{}, Status: http.StatusOK, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-delete", Use: "delete", Short: "Delete a service account", Method: http.MethodDelete, Path: "/v1/service-accounts/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-activate", Use: "activate", Short: "Activate a service account", Method: http.MethodPost, Path: "/v1/service-accounts/{id}/activate", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID}, Status: http.StatusOK, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-deactivate", Use: "deactivate", Short: "Deactivate a service account", Method: http.MethodPost, Path: "/v1/service-accounts/{id}/deactivate", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID}, Status: http.StatusOK, Destructive: true, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-reset-secret", Use: "reset-secret", Short: "Reset a service account client secret", Method: http.MethodPost, Path: "/v1/service-accounts/{id}/reset-secret", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID}, Status: http.StatusOK, ResponseBody: true, Destructive: true, SecretResponse: true, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-s3-key-attach", Use: "attach", Short: "Attach an S3 key to a service account", Method: http.MethodPost, Path: "/v1/service-accounts/{id}/s3-keys/{keyId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID, mutationPath("keyId", "s3-key-id", "S3 key ID")}, Status: http.StatusNoContent, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-s3-key-detach", Use: "detach", Short: "Detach an S3 key from a service account", Method: http.MethodDelete, Path: "/v1/service-accounts/{id}/s3-keys/{keyId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID, mutationPath("keyId", "s3-key-id", "S3 key ID")}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-swift-user-attach", Use: "attach", Short: "Attach a Swift user to a service account", Method: http.MethodPost, Path: "/v1/service-accounts/{id}/swift-users/{swiftId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID, mutationPath("swiftId", "swift-user-id", "Swift user ID")}, Status: http.StatusNoContent, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-swift-user-detach", Use: "detach", Short: "Detach a Swift user from a service account", Method: http.MethodDelete, Path: "/v1/service-accounts/{id}/swift-users/{swiftId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID, mutationPath("swiftId", "swift-user-id", "Swift user ID")}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-tag-create", Use: "create", Short: "Create tags for a service account", Method: http.MethodPost, Path: "/v1/service-accounts/{id}/tags", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID}, Body: &iamMutationBody{RequiredFields: []string{"tags"}}, Status: http.StatusCreated, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-tag-delete", Use: "delete", Short: "Delete tags from a service account", Method: http.MethodDelete, Path: "/v1/service-accounts/{id}/tags", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID}, Body: &iamMutationBody{RequiredFields: []string{"keys"}}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-trusted-root-attach", Use: "attach", Short: "Attach a trusted root to a service account", Method: http.MethodPost, Path: "/v1/service-accounts/{id}/trusted-roots/{rootPortalId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID, mutationPath("rootPortalId", "root-portal-id", "Trusted root portal ID")}, Status: http.StatusNoContent, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "service-account-trusted-root-detach", Use: "detach", Short: "Detach a trusted root from a service account", Method: http.MethodDelete, Path: "/v1/service-accounts/{id}/trusted-roots/{rootPortalId}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{serviceAccountID, mutationPath("rootPortalId", "root-portal-id", "Trusted root portal ID")}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "service-account-id",
		},
		{
			Key: "swift-user-create", Use: "create", Short: "Create a Swift user", Method: http.MethodPost, Path: "/v1/swift-users", BuildClient: createAccountsClient,
			Body: &iamMutationBody{RequiredFields: []string{"regionId", "projectId"}}, Status: http.StatusCreated, ResponseBody: true, SecretResponse: true,
		},
		{
			Key: "swift-user-update", Use: "update", Short: "Update a Swift user", Method: http.MethodPatch, Path: "/v1/swift-users/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("id", "swift-user-id", "Swift user ID")}, Body: &iamMutationBody{}, Status: http.StatusNoContent,
		},
		{
			Key: "swift-user-delete", Use: "delete", Short: "Delete a Swift user", Method: http.MethodDelete, Path: "/v1/swift-users/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{mutationPath("id", "swift-user-id", "Swift user ID")}, Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "iam-user-create", Use: "create", Short: "Create an IAM user", Method: http.MethodPost, Path: "/v1/iam-users", BuildClient: createAccountsClient,
			Body: &iamMutationBody{}, Status: http.StatusCreated, SecretInput: true,
		},
		{
			Key: "iam-user-delete", Use: "delete", Short: "Delete an IAM user", Method: http.MethodDelete, Path: "/v1/iam-users/{id}", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-activate", Use: "activate", Short: "Activate an IAM user", Method: http.MethodPost, Path: "/v1/iam-users/{id}/activate", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-activate-email", Use: "activate-email", Short: "Activate IAM-user email sign-in", Method: http.MethodPost, Path: "/v1/iam-users/{id}/activate-email", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-activate-google", Use: "activate-google", Short: "Activate IAM-user Google Authenticator", Method: http.MethodPost, Path: "/v1/iam-users/{id}/activate-google", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-deactivate", Use: "deactivate", Short: "Deactivate an IAM user", Method: http.MethodPost, Path: "/v1/iam-users/{id}/deactivate", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-deactivate-email", Use: "deactivate-email", Short: "Deactivate IAM-user email sign-in", Method: http.MethodPost, Path: "/v1/iam-users/{id}/deactivate-email", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-deactivate-google", Use: "deactivate-google", Short: "Deactivate IAM-user Google Authenticator", Method: http.MethodPost, Path: "/v1/iam-users/{id}/deactivate-google", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-reset-password", Use: "reset-password", Short: "Reset an IAM-user password", Method: http.MethodPost, Path: "/v1/iam-users/{id}/reset-password", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Body: &iamMutationBody{}, Status: http.StatusNoContent, Destructive: true, SecretInput: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-send-verification-email", Use: "send-verification-email", Short: "Send an IAM-user verification email", Method: http.MethodPost, Path: "/v1/iam-users/{id}/send-verification-email", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusNoContent, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-set-up-email", Use: "update-email", Short: "Set up IAM-user email sign-in", Method: http.MethodPost, Path: "/v1/iam-users/{id}/set-up-email", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Body: &iamMutationBody{}, Status: http.StatusNoContent, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-set-up-google", Use: "update-google", Short: "Set up IAM-user Google Authenticator", Method: http.MethodPost, Path: "/v1/iam-users/{id}/set-up-google", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Status: http.StatusOK, SelfTargetFlag: "iam-user-id", AllowUnexpectedBody: true, SecretResponse: true,
		},
		{
			Key: "iam-user-tag-create", Use: "create", Short: "Create tags for an IAM user", Method: http.MethodPost, Path: "/v1/iam-users/{id}/tags", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Body: &iamMutationBody{RequiredFields: []string{"tags"}}, Status: http.StatusCreated, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-tag-delete", Use: "delete", Short: "Delete tags from an IAM user", Method: http.MethodDelete, Path: "/v1/iam-users/{id}/tags", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Body: &iamMutationBody{RequiredFields: []string{"keys"}}, Status: http.StatusNoContent, Destructive: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-verify-email", Use: "verify-email", Short: "Verify IAM-user email sign-in", Method: http.MethodPost, Path: "/v1/iam-users/{id}/verify-email", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Body: &iamMutationBody{}, Status: http.StatusNoContent, SecretInput: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-verify-google", Use: "verify-google", Short: "Verify IAM-user Google Authenticator", Method: http.MethodPost, Path: "/v1/iam-users/{id}/verify-google", BuildClient: createAccountsClient,
			Paths: []iamMutationPath{iamUserID}, Body: &iamMutationBody{}, Status: http.StatusNoContent, SecretInput: true, SelfTargetFlag: "iam-user-id",
		},
		{
			Key: "iam-user-current-activate-email", Use: "activate-email", Short: "Activate current IAM-user email sign-in", Method: http.MethodPost, Path: "/v1/iam-users/activate-email", BuildClient: createAccountsClient,
			Status: http.StatusNoContent,
		},
		{
			Key: "iam-user-current-activate-google", Use: "activate-google", Short: "Activate current IAM-user Google Authenticator", Method: http.MethodPost, Path: "/v1/iam-users/activate-google", BuildClient: createAccountsClient,
			Status: http.StatusNoContent,
		},
		{
			Key: "iam-user-current-deactivate-email", Use: "deactivate-email", Short: "Deactivate current IAM-user email sign-in", Method: http.MethodPost, Path: "/v1/iam-users/deactivate-email", BuildClient: createAccountsClient,
			Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "iam-user-current-deactivate-google", Use: "deactivate-google", Short: "Deactivate current IAM-user Google Authenticator", Method: http.MethodPost, Path: "/v1/iam-users/deactivate-google", BuildClient: createAccountsClient,
			Status: http.StatusNoContent, Destructive: true,
		},
		{
			Key: "iam-user-current-reset-password", Use: "reset-password", Short: "Reset the current IAM-user password", Method: http.MethodPost, Path: "/v1/iam-users/reset-password", BuildClient: createAccountsClient,
			Body: &iamMutationBody{}, Status: http.StatusNoContent, Destructive: true, SecretInput: true,
		},
		{
			Key: "iam-user-current-send-verification-email", Use: "send-verification-email", Short: "Send a verification email for the current IAM user", Method: http.MethodPost, Path: "/v1/iam-users/send-verification-email", BuildClient: createAccountsClient,
			Status: http.StatusNoContent,
		},
		{
			Key: "iam-user-current-set-up-email", Use: "update-email", Short: "Set up email sign-in for the current IAM user", Method: http.MethodPost, Path: "/v1/iam-users/set-up-email", BuildClient: createAccountsClient,
			Body: &iamMutationBody{}, Status: http.StatusNoContent,
		},
		{
			Key: "iam-user-current-set-up-google", Use: "update-google", Short: "Set up Google Authenticator for the current IAM user", Method: http.MethodPost, Path: "/v1/iam-users/set-up-google", BuildClient: createAccountsClient,
			Status: http.StatusOK, AllowUnexpectedBody: true, SecretResponse: true,
		},
		{
			Key: "iam-user-current-verify-email", Use: "verify-email", Short: "Verify email sign-in for the current IAM user", Method: http.MethodPost, Path: "/v1/iam-users/verify-email", BuildClient: createAccountsClient,
			Body: &iamMutationBody{}, Status: http.StatusNoContent, SecretInput: true,
		},
	}
}
