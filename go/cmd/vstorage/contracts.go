package vstorage

import "net/http"

var (
	projectID      = pathParameter{Placeholder: "project_id", Flag: "project-id", Usage: "vStorage project ID"}
	regionID       = pathParameter{Placeholder: "region_id", Flag: "region-id", Usage: "vStorage region ID"}
	userID         = pathParameter{Placeholder: "user_id", Flag: "user-id", Usage: "vStorage user ID"}
	container      = pathParameter{Placeholder: "container", Flag: "container", Usage: "HCM03 container name", Validate: validateResourceName("container")}
	bucket         = pathParameter{Placeholder: "bucket", Flag: "bucket", Usage: "HAN02/HCM04 bucket name", Validate: validateResourceName("bucket")}
	object         = pathParameter{Placeholder: "object", Flag: "object", Usage: "Object key", Validate: validateObjectName}
	directory      = pathParameter{Placeholder: "directory", Flag: "directory", Usage: "Directory name", Validate: validateObjectName}
	swiftObject    = pathParameter{Placeholder: "object", Flag: "object", Usage: "HCM03 object key", Validate: validateSwiftObjectName}
	swiftDirectory = pathParameter{Placeholder: "directory", Flag: "directory", Usage: "HCM03 directory name", Validate: validateSwiftObjectName}
	ruleName       = pathParameter{Placeholder: "rule_name", Flag: "rule-name", Usage: "Lifecycle rule name", Validate: validateLifecycleRuleName}
	noticeID       = pathParameter{Placeholder: "id", Flag: "notification-id", Usage: "Notification ID"}
	versionID      = pathParameter{Placeholder: "version_id", Flag: "version-id", Usage: "Object version ID"}
)

var (
	startTime      = queryParameter{WireName: "startTime", Flag: "start-time", Usage: "Range start time in the API format", Required: true}
	endTime        = queryParameter{WireName: "endTime", Flag: "end-time", Usage: "Range end time in the API format", Required: true}
	versionIDQuery = queryParameter{WireName: "versionId", Flag: "version-id", Usage: "Optional opaque object version ID"}
)

func allOperations() []operation {
	operations := make([]operation, 0, 97)
	operations = append(operations, commonOperations()...)
	operations = append(operations, swiftOperations()...)
	operations = append(operations, cephOperations()...)
	return operations
}

func commonOperations() []operation {
	return []operation{
		{Parent: "region", Use: "list", Short: "List vStorage regions", Method: http.MethodGet, Path: "/api/v1/regions"},
		{Parent: "region", Use: "get", Short: "Get a vStorage region", Method: http.MethodGet, Path: "/api/v1/regions/{region_id}", Paths: []pathParameter{regionID}},
		{Parent: "project", Use: "list", Short: "List vStorage projects", Method: http.MethodGet, Path: "/api/v1/projects"},
		{Parent: "project", Use: "get", Short: "Get a vStorage project", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/details", Paths: []pathParameter{projectID}},
		{Parent: "project", Use: "create", Short: "Create a vStorage project", Method: http.MethodPost, Path: "/api/v1/projects", Body: requiredObjectBody, Extra: vstorageExtra{PoCField: "isPoc"}, Mutation: true},
		{Parent: "project", Use: "resize", Short: "Resize a vStorage project quota", Method: http.MethodPut, Path: "/api/v1/projects/{project_id}/resize", Paths: []pathParameter{projectID}, Body: requiredObjectBody, Mutation: true},
		{Parent: "billing", Use: "get-request-summary", Short: "Get project request statistics for a time range", Method: http.MethodGet, Path: "/api/v1/billing/statistics/projects/{project_id}/request", Paths: []pathParameter{projectID}, Queries: []queryParameter{startTime, endTime}},
		{Parent: "billing", Use: "get-traffic-summary", Short: "Get project traffic statistics for a time range", Method: http.MethodGet, Path: "/api/v1/billing/statistics/projects/{project_id}/traffic", Paths: []pathParameter{projectID}, Queries: []queryParameter{startTime, endTime}},
		{Parent: "billing", Use: "get-usage-summary", Short: "Get project usage statistics for a time range", Method: http.MethodGet, Path: "/api/v1/billing/statistics/projects/{project_id}/usage", Paths: []pathParameter{projectID}, Queries: []queryParameter{startTime, endTime}},
		{Parent: "billing", Use: "get-quota", Short: "Get the current project quota", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/quota", Paths: []pathParameter{projectID}},
		{Parent: "billing", Use: "get-traffic", Short: "Get the current project traffic", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/traffic", Paths: []pathParameter{projectID}},
		{Parent: "billing", Use: "get-traffic-range", Short: "Get project traffic for a time range", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/traffic/search", Paths: []pathParameter{projectID}, Queries: []queryParameter{startTime, endTime}},
		{Parent: "billing", Use: "get-usage", Short: "Get the current project usage", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/usage", Paths: []pathParameter{projectID}},
		{Parent: "billing", Use: "get-usage-range", Short: "Get project usage for a time range", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/usage/search", Paths: []pathParameter{projectID}, Queries: []queryParameter{startTime, endTime}},
		{Parent: "billing", Use: "get-user-usage", Short: "Get usage across a user's projects for a time range", Method: http.MethodGet, Path: "/api/v1/users/{user_id}/usage", Paths: []pathParameter{userID}, Queries: []queryParameter{{WireName: "start_time", Flag: "start-time", Usage: "Range start time in the API format", Required: true}, {WireName: "end_time", Flag: "end-time", Usage: "Range end time in the API format", Required: true}}},
	}
}

func swiftOperations() []operation {
	operations := []operation{
		{Parent: "project", Use: "delete", Short: "Delete an HCM03 vStorage project and all of its data", Method: http.MethodDelete, Path: "/api/v1/projects/{project_id}", Paths: []pathParameter{projectID}, Mutation: true, Destructive: true},
		{Parent: "project", Use: "list-members", Short: "List members of an HCM03 vStorage project", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/members", Paths: []pathParameter{projectID}},

		{Parent: "container", Use: "list", Short: "List HCM03 containers", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}", Paths: []pathParameter{projectID}, Queries: []queryParameter{{WireName: "limit", Flag: "limit", Usage: "Maximum number of containers"}, {WireName: "marker", Flag: "marker", Usage: "Pagination marker"}, {WireName: "permission", Flag: "permission", Usage: "Permission filter"}}},
		{Parent: "container", Use: "create", Short: "Create an HCM03 container", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}", Paths: []pathParameter{projectID, container}, Mutation: true},
		{Parent: "container", Use: "delete", Short: "Delete an HCM03 container", Method: http.MethodDelete, Path: "/api/v1/projects/{project_id}/containers/{container}", Paths: []pathParameter{projectID, container}, Mutation: true, Destructive: true},
		{Parent: "container", Use: "get", Short: "Get HCM03 container details", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/{container}/details", Paths: []pathParameter{projectID, container}},
		{Parent: "container", Use: "get-acl", Short: "Get an HCM03 container ACL", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}/acls", Paths: []pathParameter{projectID, container}},
		{Parent: "container", Use: "update-acl", Short: "Replace an HCM03 container ACL", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/acls", Paths: []pathParameter{projectID, container}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container", Use: "delete-cors", Short: "Delete HCM03 container CORS configuration", Method: http.MethodDelete, Path: "/api/v1/projects/{project_id}/containers/{container}/cors", Paths: []pathParameter{projectID, container}, Mutation: true, Destructive: true},
		{Parent: "container", Use: "update-cors", Short: "Update HCM03 container CORS configuration", Method: http.MethodPut, Path: "/api/v1/projects/{project_id}/containers/{container}/cors", Paths: []pathParameter{projectID, container}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container", Use: "get-public-access", Short: "Get HCM03 container public-access configuration", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}/public_access", Paths: []pathParameter{projectID, container}},
		{Parent: "container", Use: "update-public-access", Short: "Update HCM03 container public-access configuration", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/public_access", Paths: []pathParameter{projectID, container}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container", Use: "delete-static-web", Short: "Disable static website hosting for an HCM03 container", Method: http.MethodDelete, Path: "/api/v1/projects/{project_id}/containers/{container}/static_web", Paths: []pathParameter{projectID, container}, Mutation: true, Destructive: true},
		{Parent: "container", Use: "update-static-web", Short: "Enable static website hosting for an HCM03 container", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/static_web", Paths: []pathParameter{projectID, container}, Mutation: true},
		{Parent: "container", Use: "get-url", Short: "Get an HCM03 container storage URL", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}/url", Paths: []pathParameter{projectID, container}},
		{Parent: "container", Use: "search", Short: "Search HCM03 containers by name", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers_searching", Paths: []pathParameter{projectID}, Body: requiredObjectBody},

		{Parent: "container/object", Use: "list", Short: "List objects in an HCM03 container or directory", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}", Paths: []pathParameter{projectID, container}, Queries: []queryParameter{{WireName: "directoryName", Flag: "directory-name", Usage: "Directory name filter"}, {WireName: "limit", Flag: "limit", Usage: "Maximum number of objects"}, {WireName: "marker", Flag: "marker", Usage: "Pagination marker"}}},
		{Parent: "container/object", Use: "list-directories", Short: "List directories in an HCM03 container", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}/directories", Paths: []pathParameter{projectID, container}},
		{Parent: "container/object", Use: "create-directory", Short: "Create a directory in an HCM03 container", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/directories", Paths: []pathParameter{projectID, container}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container/object", Use: "delete-directory", Short: "Delete a directory from an HCM03 container", Method: http.MethodDelete, Path: "/api/v1/projects/{project_id}/containers/{container}/directories/{directory}", Paths: []pathParameter{projectID, container, swiftDirectory}, Mutation: true, Destructive: true},
		{Parent: "container/object", Use: "generate-directory-download-url", Short: "Generate an HCM03 directory download URL", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/directories/{directory}/download_tempurls", Paths: []pathParameter{projectID, container, swiftDirectory}, Queries: []queryParameter{{WireName: "expiredTime", Flag: "expired-time", Usage: "URL expiry in the API format"}, {WireName: "viewMode", Flag: "view-mode", Usage: "Directory view mode"}}, Mutation: true},
		{Parent: "container/object", Use: "generate-directory-upload-url", Short: "Generate an HCM03 directory upload URL", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/directories/{directory}/upload_tempurls", Paths: []pathParameter{projectID, container, swiftDirectory}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container/object", Use: "search", Short: "Search objects in an HCM03 container", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/search", Paths: []pathParameter{projectID, container}, Body: requiredObjectBody},
		{Parent: "container/object", Use: "delete", Short: "Delete an object from an HCM03 container", Method: http.MethodDelete, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}", Paths: []pathParameter{projectID, container, swiftObject}, Mutation: true, Destructive: true},
		{Parent: "container/object", Use: "get", Short: "Get HCM03 object details", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}/details", Paths: []pathParameter{projectID, container, swiftObject}},
		{Parent: "container/object", Use: "generate-download-url", Short: "Generate an HCM03 object download URL", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}/download_tempurls", Paths: []pathParameter{projectID, container, swiftObject}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container/object", Use: "get-metadata", Short: "Get HCM03 object metadata", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}/metadata", Paths: []pathParameter{projectID, container, swiftObject}},
		{Parent: "container/object", Use: "update-metadata", Short: "Update HCM03 object metadata", Method: http.MethodPut, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}/metadata", Paths: []pathParameter{projectID, container, swiftObject}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container/object", Use: "get-tags", Short: "Get HCM03 object tags", Method: http.MethodGet, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}/tags", Paths: []pathParameter{projectID, container, swiftObject}},
		{Parent: "container/object", Use: "update-tags", Short: "Update HCM03 object tags", Method: http.MethodPut, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}/tags", Paths: []pathParameter{projectID, container, swiftObject}, Body: requiredObjectBody, Mutation: true},
		{Parent: "container/object", Use: "generate-upload-url", Short: "Generate an HCM03 object upload URL", Method: http.MethodPost, Path: "/api/v1/projects/{project_id}/containers/{container}/objects/{object}/upload_tempurls", Paths: []pathParameter{projectID, container, swiftObject}, Body: requiredObjectBody, Mutation: true},
	}
	for i := range operations {
		operations[i].Extra = vstorageExtra{Family: hcm03Family}
	}
	return operations
}
