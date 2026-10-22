package vcr

import (
	"fmt"
	"net/http"

	opengine "github.com/greennodehub/greennode-cli/internal/operation"
)

var (
	repositoryID = pathParameter{Placeholder: "repoId", Flag: "repository-id", Usage: "vCR repository ID"}
	userID       = pathParameter{Placeholder: "repoUserId", Flag: "user-id", Usage: "vCR repository user ID"}
)

var (
	imageName = queryParameter{WireName: "imageName", Flag: "image-name", Usage: "Image name", Required: true}
	digest    = queryParameter{WireName: "digest", Flag: "digest", Usage: "Artifact digest", Required: true}
	name      = queryParameter{WireName: "name", Flag: "name", Usage: "Filter by name"}
	page      = queryParameter{WireName: "page", Flag: "page", Usage: "Page number (starts at 1)", Kind: queryInteger, Minimum: 1}
	size      = queryParameter{WireName: "size", Flag: "size", Usage: "Page size (must be at least 1)", Kind: queryInteger, Minimum: 1}
)

func objectBody(name string, requiredFields ...string) *bodyContract {
	return &bodyContract{
		Kind:           opengine.ObjectBody,
		Style:          opengine.BodyStyleJSONObject,
		Usage:          fmt.Sprintf("Request body as a %s JSON object", name),
		Name:           name,
		RequiredFields: requiredFields,
	}
}

func allOperations() []operation {
	return []operation{
		{Parent: "artifact", Use: "list", Short: "List artifacts for a repository image", OperationID: "listArtifactUsingGET_1", Method: http.MethodGet, Path: "/v1/repository/{repoId}/images/artifacts", Paths: []pathParameter{repositoryID}, Queries: []queryParameter{imageName, name, page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "artifact", Use: "delete", Short: "Delete an image artifact", OperationID: "deleteImageUsingDELETE_2", Method: http.MethodDelete, Path: "/v1/repository/{repoId}/images/artifacts/delete", Paths: []pathParameter{repositoryID}, Queries: []queryParameter{digest, imageName}, Status: http.StatusOK, Mutation: true, Destructive: true},

		{Parent: "image", Use: "list", Short: "List images in a repository", OperationID: "listImageUsingGET_3", Method: http.MethodGet, Path: "/v1/repository/{repoId}/images", Paths: []pathParameter{repositoryID}, Queries: []queryParameter{name, page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "image", Use: "delete", Short: "Delete a repository image", OperationID: "deleteImageUsingDELETE_3", Method: http.MethodDelete, Path: "/v1/repository/{repoId}/images/delete", Paths: []pathParameter{repositoryID}, Queries: []queryParameter{imageName}, Status: http.StatusOK, Mutation: true, Destructive: true},
		{Parent: "image", Use: "get", Short: "Get repository image details", OperationID: "listImageUsingGET_2", Method: http.MethodGet, Path: "/v1/repository/{repoId}/images/detail", Paths: []pathParameter{repositoryID}, Queries: []queryParameter{imageName}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "repository", Use: "list", Short: "List repositories available to the current user", OperationID: "listRepoUsingGET_3", Method: http.MethodGet, Path: "/v1/repository", Queries: []queryParameter{{WireName: "accessLevel", Flag: "access-level", Usage: "Filter by access level"}, name, page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "repository", Use: "create", Short: "Create a repository", OperationID: "createRepoUsingPOST_1", Method: http.MethodPost, Path: "/v1/repository", Body: objectBody("CreateRepoRequest", "isPublic", "quotaLimit", "repoName"), Status: http.StatusAccepted, ResponseBody: true, Mutation: true},
		{Parent: "repository", Use: "delete", Short: "Delete a repository", OperationID: "deleteRepoUsingDELETE_1", Method: http.MethodDelete, Path: "/v1/repository/{repoId}", Paths: []pathParameter{repositoryID}, Status: http.StatusAccepted, ResponseBody: true, Mutation: true, Destructive: true},
		{Parent: "repository", Use: "get", Short: "Get repository details", OperationID: "listRepoUsingGET_2", Method: http.MethodGet, Path: "/v1/repository/{repoId}", Paths: []pathParameter{repositoryID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "repository", Use: "attach-users", Short: "Attach repository users to a repository", OperationID: "attachUserUsingPUT_1", Method: http.MethodPut, Path: "/v1/repository/{repoId}/attach", Paths: []pathParameter{repositoryID}, Body: objectBody("AttachRepoUserRequest", "repoId", "repoUserList"), Extra: []bodyBinding{{Field: "repoId", Flag: "repository-id"}}, Status: http.StatusOK, Mutation: true},
		{Parent: "repository", Use: "detach-users", Short: "Detach repository users from a repository", OperationID: "detachUserUsingPUT_1", Method: http.MethodPut, Path: "/v1/repository/{repoId}/detach", Paths: []pathParameter{repositoryID}, Body: objectBody("DetachRepoUserRequest", "repoId", "repoUserUuidList"), Extra: []bodyBinding{{Field: "repoId", Flag: "repository-id"}}, Status: http.StatusOK, Mutation: true},
		{Parent: "repository", Use: "list-history", Short: "List repository history", OperationID: "getRepoHistoryUsingGET_1", Method: http.MethodGet, Path: "/v1/repository/{repoId}/history", Paths: []pathParameter{repositoryID}, Queries: []queryParameter{page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "repository", Use: "update-quota", Short: "Update repository quota", OperationID: "updateRepoQuotaLimitUsingPUT_1", Method: http.MethodPut, Path: "/v1/repository/{repoId}/quotas", Paths: []pathParameter{repositoryID}, Body: objectBody("UpdateRepoQuotaRequest", "quotaLimit", "repoId"), Extra: []bodyBinding{{Field: "repoId", Flag: "repository-id"}}, Status: http.StatusOK, ResponseBody: true, Mutation: true},
		{Parent: "repository", Use: "list-users", Short: "List users attached to a repository", OperationID: "getUserAttachedRepoUsingGET_1", Method: http.MethodGet, Path: "/v1/repository/{repoId}/user", Paths: []pathParameter{repositoryID}, Queries: []queryParameter{name, page, size}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "user", Use: "list", Short: "List repository users", OperationID: "listRepoUserUsingGET_1", Method: http.MethodGet, Path: "/v1/user", Queries: []queryParameter{name, page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "user", Use: "create", Short: "Create a repository user", OperationID: "createRepoUserUsingPOST_1", Method: http.MethodPost, Path: "/v1/user", Body: objectBody("CreateRepoUserRequest", "name", "permissionRequestList"), Status: http.StatusOK, ResponseBody: true, Mutation: true, SecretResponse: true},
		{Parent: "user", Use: "list-permissions", Short: "List repository-user permissions", OperationID: "getPermissionListUsingGET_1", Method: http.MethodGet, Path: "/v1/user/permissions", Status: http.StatusOK, ResponseBody: true},
		{Parent: "user", Use: "delete", Short: "Delete a repository user", OperationID: "deleteRepoUserUsingDELETE_1", Method: http.MethodDelete, Path: "/v1/user/{repoUserId}", Paths: []pathParameter{userID}, Status: http.StatusOK, Mutation: true, Destructive: true},
		{Parent: "user", Use: "update", Short: "Update a repository user", OperationID: "updateRepoUserUsingPUT_1", Method: http.MethodPut, Path: "/v1/user/{repoUserId}", Paths: []pathParameter{userID}, Body: objectBody("UpdateRepoUserRequest", "repoUserId"), Extra: []bodyBinding{{Field: "repoUserId", Flag: "user-id"}}, Status: http.StatusOK, ResponseBody: true, Mutation: true},
		{Parent: "user", Use: "disable", Short: "Disable a repository user", OperationID: "disableRepoUserUsingPUT_1", Method: http.MethodPut, Path: "/v1/user/{repoUserId}/disable", Paths: []pathParameter{userID}, Status: http.StatusOK, ResponseBody: true, Mutation: true},
		{Parent: "user", Use: "enable", Short: "Enable a repository user", OperationID: "enableRepoUserUsingPUT_1", Method: http.MethodPut, Path: "/v1/user/{repoUserId}/enable", Paths: []pathParameter{userID}, Status: http.StatusOK, ResponseBody: true, Mutation: true},
		{Parent: "user", Use: "update-permissions", Short: "Update repository-user permissions", OperationID: "updateRepoUserPermissionUsingPUT_1", Method: http.MethodPut, Path: "/v1/user/{repoUserId}/permission", Paths: []pathParameter{userID}, Body: objectBody("UpdatePermissionRepoUserRequest", "permissionRequestList", "repoUserId"), Extra: []bodyBinding{{Field: "repoUserId", Flag: "user-id"}}, Status: http.StatusOK, ResponseBody: true, Mutation: true},
		{Parent: "user", Use: "refresh-secret", Short: "Rotate and return a repository-user secret", OperationID: "refreshRepoUserUsingGET_1", Method: http.MethodGet, Path: "/v1/user/{repoUserId}/refresh", Paths: []pathParameter{userID}, Status: http.StatusOK, ResponseBody: true, Mutation: true, Destructive: true, SecretResponse: true, RawResponse: true},
	}
}
