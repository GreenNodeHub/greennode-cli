package vmonitorlog

import "net/http"

func allOperations() []operation {
	return []operation{
		{Parent: "archive", Use: "list", Short: "List archives", Method: http.MethodGet, Path: "/v1/archives", Queries: queries("query", "project_id", "page", "size"), SecretResponse: true},
		{Parent: "archive", Use: "create", Short: "Create an archive", Method: http.MethodPost, Path: "/v1/archives", Body: requiredObjectBody, Mutation: true, SecretResponse: true},
		{Parent: "archive", Use: "test-connection", Short: "Test an archive storage connection", Method: http.MethodPost, Path: "/v1/archives/test-connection", Body: requiredObjectBody},
		{Parent: "archive", Use: "delete", Short: "Delete an archive", Method: http.MethodDelete, Path: "/v1/archives/{archive_id}", Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusNoContent}},
		{Parent: "archive", Use: "get", Short: "Get an archive", Method: http.MethodGet, Path: "/v1/archives/{archive_id}", SecretResponse: true},
		{Parent: "archive", Use: "update", Short: "Update an archive", Method: http.MethodPut, Path: "/v1/archives/{archive_id}", Body: requiredObjectBody, Mutation: true, SecretResponse: true},

		{Parent: "certificate", Use: "download", Short: "Download a project certificate bundle", Method: http.MethodGet, Path: "/v1/downloads/certificates/projects/{project_id}/{cert_id}", Download: true},

		{Parent: "log", Use: "get-data-exists", Short: "Check whether a project contains log data", Method: http.MethodGet, Path: "/v1/projects/{projectId}/exists-log-data"},
		{Parent: "log", Use: "create-export", Short: "Create a log export job", Method: http.MethodPost, Path: "/v1/projects/{projectId}/log-exports", Body: requiredObjectBody, Mutation: true},
		{Parent: "log", Use: "get-export", Short: "Get a log export job", Method: http.MethodGet, Path: "/v1/projects/{projectId}/log-exports/{id}"},
		{Parent: "log", Use: "search", Short: "Search logs in a project", Method: http.MethodPost, Path: "/v1/projects/{projectId}/search-logs", Body: requiredObjectBody},
		{Parent: "log", Use: "search-default", Short: "Search default audit logs in a project", Method: http.MethodPost, Path: "/v1/projects/{projectId}/search-logs/default", Body: requiredObjectBody},

		{Parent: "pipeline", Use: "list", Short: "List pipelines", Method: http.MethodGet, Path: "/v1/pipelines", Queries: queries("query", "page", "size")},
		{Parent: "pipeline", Use: "create", Short: "Create a pipeline", Method: http.MethodPost, Path: "/v1/pipelines", Body: requiredObjectBody, Mutation: true},
		{Parent: "pipeline", Use: "get", Short: "Get a pipeline", Method: http.MethodGet, Path: "/v1/pipelines/{id}"},
		{Parent: "pipeline", Use: "delete", Short: "Delete a pipeline", Method: http.MethodDelete, Path: "/v1/pipelines/{pipeline_id}", Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusNoContent}},
		{Parent: "pipeline", Use: "update", Short: "Update a pipeline", Method: http.MethodPut, Path: "/v1/pipelines/{pipeline_id}", Body: requiredObjectBody, Mutation: true},

		{Parent: "processor", Use: "debug-grok", Short: "Debug a GROK pattern", Method: http.MethodPost, Path: "/v1/processors/debug-grok-parser", Body: requiredObjectBody},
		{Parent: "processor", Use: "list-date-formats", Short: "List supported date formats", Method: http.MethodGet, Path: "/v1/processors/formats-date"},
		{Parent: "processor", Use: "create", Short: "Create a processor", Method: http.MethodPost, Path: "/v1/processors/{pipeline_id}/{processor_group_id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "processor", Use: "delete", Short: "Delete a processor", Method: http.MethodDelete, Path: "/v1/processors/{pipeline_id}/{processor_group_id}/{processor_id}", Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusAccepted}},
		{Parent: "processor", Use: "update", Short: "Update a processor", Method: http.MethodPut, Path: "/v1/processors/{pipeline_id}/{processor_group_id}/{processor_id}", Body: requiredObjectBody, Mutation: true},

		{Parent: "processor-group", Use: "create", Short: "Create a processor group", Method: http.MethodPost, Path: "/v1/processor-groups/{pipeline_id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "processor-group", Use: "delete", Short: "Delete a processor group", Method: http.MethodDelete, Path: "/v1/processor-groups/{pipeline_id}/{processor_group_id}", Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusAccepted}},
		{Parent: "processor-group", Use: "get", Short: "Get a processor group", Method: http.MethodGet, Path: "/v1/processor-groups/{pipeline_id}/{processor_group_id}"},
		{Parent: "processor-group", Use: "update", Short: "Update a processor group", Method: http.MethodPut, Path: "/v1/processor-groups/{pipeline_id}/{processor_group_id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "processor-group", Use: "reorder", Short: "Reorder processors in a group", Method: http.MethodPut, Path: "/v1/processor-groups/{pipeline_id}/{processor_group_id}/re-order", Body: requiredObjectBody, Mutation: true, EmptySuccess: []int{http.StatusAccepted}},

		{Parent: "processor-group-library", Use: "list", Short: "List processor group libraries", Method: http.MethodGet, Path: "/v1/processor-group-libraries", Queries: queries("query")},
		{Parent: "processor-group-library", Use: "clone", Short: "Clone a processor group library", Method: http.MethodPost, Path: "/v1/processor-group-libraries", Body: requiredObjectBody, Mutation: true},

		{Parent: "project", Use: "list", Short: "List log projects", Method: http.MethodGet, Path: "/v1/projects", Queries: queries("query", "page", "size", "billing_status", "project_type", "status")},
		{Parent: "project", Use: "get-topic-info", Short: "Get a project's topic information", Method: http.MethodGet, Path: "/v1/projects/topic-info/{id}"},
		{Parent: "project", Use: "get", Short: "Get a log project", Method: http.MethodGet, Path: "/v1/projects/{id}"},
		{Parent: "project", Use: "create-certificate", Short: "Create a project certificate", Method: http.MethodPost, Path: "/v1/projects/{id}/certificates", Mutation: true, EmptySuccess: []int{http.StatusNoContent}},
		{Parent: "project", Use: "update", Short: "Update a log project", Method: http.MethodPatch, Path: "/v1/projects/{project_id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "project", Use: "delete-certificate", Short: "Delete a project certificate", Method: http.MethodDelete, Path: "/v1/projects/{project_id}/certificates/{cert_id}", Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusNoContent}},
		{Parent: "project", Use: "list-mappings", Short: "List a project's field mappings", Method: http.MethodGet, Path: "/v1/projects/{project_id}/mappings", Queries: queries("refresh")},
		{Parent: "project", Use: "update-mappings", Short: "Update a project's field mapping format", Method: http.MethodPut, Path: "/v1/projects/{project_id}/mappings", Body: requiredObjectBody, Mutation: true, EmptySuccess: []int{http.StatusNoContent}},

		{Parent: "refill", Use: "list", Short: "List refills", Method: http.MethodGet, Path: "/v1/refills", Queries: append(requiredQueries("project_id"), queries("query", "page", "size")...), SecretResponse: true},
		{Parent: "refill", Use: "create", Short: "Create a refill from external storage", Method: http.MethodPost, Path: "/v1/refills", Body: requiredObjectBody, Mutation: true, SecretResponse: true},
		{Parent: "refill", Use: "create-from-archive", Short: "Create a refill from an archive", Method: http.MethodPost, Path: "/v1/refills/collections", Body: requiredObjectBody, Mutation: true, SecretResponse: true},
		{Parent: "refill", Use: "test-connection", Short: "Test a refill storage connection", Method: http.MethodPost, Path: "/v1/refills/test-connection", Body: requiredObjectBody},
		{Parent: "refill", Use: "delete", Short: "Delete a refill", Method: http.MethodDelete, Path: "/v1/refills/{refill_id}", Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusNoContent}},
		{Parent: "refill", Use: "get", Short: "Get a refill", Method: http.MethodGet, Path: "/v1/refills/{refill_id}", SecretResponse: true},

		{Parent: "vcdn-mapping", Use: "list", Short: "List vCDN log mappings", Method: http.MethodGet, Path: "/v1/vcdn-log-mapping", Queries: queries("query", "sortBy", "sortOrder", "page", "type", "size")},
		{Parent: "vcdn-mapping", Use: "disable", Short: "Disable a vCDN log mapping", Method: http.MethodPatch, Path: "/v1/vcdn-log-mapping/disable/{cdn-domain}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vcdn-mapping", Use: "update", Short: "Update a vCDN log mapping", Method: http.MethodPatch, Path: "/v1/vcdn-log-mapping/edit/{cdn-domain}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vcdn-mapping", Use: "enable", Short: "Enable a vCDN log mapping", Method: http.MethodPatch, Path: "/v1/vcdn-log-mapping/enable/{cdn-domain}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vcdn-mapping", Use: "list-types", Short: "List vCDN log mapping types", Method: http.MethodGet, Path: "/v1/vcdn-log-mapping/type"},

		{Parent: "vdb-mapping", Use: "list", Short: "List vDB log mappings", Method: http.MethodGet, Path: "/v1/vdb-log-mapping", Queries: queries("query", "sortBy", "sortOrder", "page", "size", "region")},
		{Parent: "vdb-mapping", Use: "disable", Short: "Disable a vDB log mapping", Method: http.MethodPatch, Path: "/v1/vdb-log-mapping/disable/{vdb-resource-id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vdb-mapping", Use: "update", Short: "Update a vDB log mapping", Method: http.MethodPatch, Path: "/v1/vdb-log-mapping/edit/{vdb-resource-id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vdb-mapping", Use: "enable", Short: "Enable a vDB log mapping", Method: http.MethodPatch, Path: "/v1/vdb-log-mapping/enable/{vdb-resource-id}", Body: requiredObjectBody, Mutation: true},

		{Parent: "vlb-mapping", Use: "list", Short: "List vLB log mappings", Method: http.MethodGet, Path: "/v1/vlb-log-mapping", Queries: queries("query", "sortBy", "sortOrder", "page", "size", "region")},
		{Parent: "vlb-mapping", Use: "disable", Short: "Disable a vLB log mapping", Method: http.MethodPatch, Path: "/v1/vlb-log-mapping/disable/{vlb-project-id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vlb-mapping", Use: "update", Short: "Update a vLB log mapping", Method: http.MethodPatch, Path: "/v1/vlb-log-mapping/edit/{vlb-project-id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vlb-mapping", Use: "enable", Short: "Enable a vLB log mapping", Method: http.MethodPatch, Path: "/v1/vlb-log-mapping/enable/{vlb-project-id}", Body: requiredObjectBody, Mutation: true},

		{Parent: "vstorage-bucket-mapping", Use: "list", Short: "List vStorage bucket log mappings", Method: http.MethodGet, Path: "/v1/vstorage-bucket-log-mappings", Queries: queries("query", "sortBy", "sortOrder", "region-id", "page", "size")},
		{Parent: "vstorage-bucket-mapping", Use: "update", Short: "Update a vStorage bucket log mapping", Method: http.MethodPatch, Path: "/v1/vstorage-bucket-log-mappings/{bucket-name}", Body: requiredObjectBody, Mutation: true},

		{Parent: "vstorage-mapping", Use: "list", Short: "List vStorage log mappings", Method: http.MethodGet, Path: "/v1/vstorage-log-mappings", Queries: queries("query", "sortBy", "sortOrder", "region-id", "page", "size")},
		{Parent: "vstorage-mapping", Use: "disable", Short: "Disable a vStorage log mapping", Method: http.MethodPatch, Path: "/v1/vstorage-log-mappings/disable/{vstorage-project-id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vstorage-mapping", Use: "update", Short: "Update a vStorage log mapping", Method: http.MethodPatch, Path: "/v1/vstorage-log-mappings/edit/{vstorage-project-id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vstorage-mapping", Use: "enable", Short: "Enable a vStorage log mapping", Method: http.MethodPatch, Path: "/v1/vstorage-log-mappings/enable/{vstorage-project-id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "vstorage-mapping", Use: "list-regions", Short: "List vStorage log mapping regions", Method: http.MethodGet, Path: "/v1/vstorage-log-mappings/regions"},
	}
}

func queries(names ...string) []queryParameter {
	parameters := make([]queryParameter, len(names))
	for i, name := range names {
		parameters[i] = queryParameter{WireName: name, Flag: flagName(name), Usage: "Query parameter " + name}
		if name == "query" {
			parameters[i].Flag = "search"
		}
	}
	return parameters
}

func requiredQueries(names ...string) []queryParameter {
	parameters := queries(names...)
	for i := range parameters {
		parameters[i].Required = true
	}
	return parameters
}
