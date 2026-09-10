package vmonitor

import "net/http"

func allOperations() []operation {
	operations := []operation{
		{Parent: "alarm", Use: "list", Method: http.MethodGet, Path: "/api/v1/alarms/list", Queries: queries("name", "page", "severity", "size", "status", "type-alarm")},
		{Parent: "alarm", Use: "create-log", Method: http.MethodPost, Path: "/api/v1/alarms/logs", Body: requiredObjectBody, Mutation: true},
		{Parent: "alarm", Use: "get-log-history", Method: http.MethodGet, Path: "/api/v1/alarms/logs/butler/{alarmId}/histories", Queries: queries("end", "len", "order", "page", "start")},
		{Parent: "alarm", Use: "get-log-status", Method: http.MethodGet, Path: "/api/v1/alarms/logs/butler/{alarmId}/status"},
		{Parent: "alarm", Use: "update-log", Method: http.MethodPut, Path: "/api/v1/alarms/logs/{alarmId}", Body: requiredObjectBody, Mutation: true},
		{Parent: "alarm", Use: "delete-log", Method: http.MethodDelete, Path: "/api/v1/alarms/logs/{alarm_id}", Mutation: true, Destructive: true},
		{Parent: "alarm", Use: "get-mapping", Method: http.MethodGet, Path: "/api/v1/alarms/mapping/{mappingId}"},
		{Parent: "alarm", Use: "create-metric", Method: http.MethodPost, Path: "/api/v1/alarms/metrics", Body: requiredObjectBody, Mutation: true},
		{Parent: "alarm", Use: "get-metric", Method: http.MethodGet, Path: "/api/v1/alarms/metrics/mona/{alarmId}"},
		{Parent: "alarm", Use: "get-metric-history", Method: http.MethodGet, Path: "/api/v1/alarms/metrics/mona/{alarmId}/histories", Queries: queries("end_time", "interval", "start_time")},
		{Parent: "alarm", Use: "delete-metric-sub-alarm", Method: http.MethodDelete, Path: "/api/v1/alarms/metrics/sub-alarms/{alarm_id}", Mutation: true, Destructive: true},
		{Parent: "alarm", Use: "update-metric", Method: http.MethodPut, Path: "/api/v1/alarms/metrics/{alarmId}", Body: requiredObjectBody, Mutation: true},
		{Parent: "alarm", Use: "delete-metric", Method: http.MethodDelete, Path: "/api/v1/alarms/metrics/{alarm_id}", Mutation: true, Destructive: true},
		{Parent: "alarm", Use: "get-synthetic-metric", Method: http.MethodGet, Path: "/api/v1/alarms/synthetic/metrics/mona/{alarmId}"},
		{Parent: "alarm", Use: "get-synthetic-metric-history", Method: http.MethodGet, Path: "/api/v1/alarms/synthetic/metrics/mona/{alarmId}/histories", Queries: queries("end_time", "interval", "start_time")},
		{Parent: "alarm", Use: "get", Method: http.MethodGet, Path: "/api/v1/alarms/{alarmId}"},

		{Parent: "api-key", Use: "create-metric", Method: http.MethodPost, Path: "/api/v1/apikeys/metric", Body: requiredObjectBody, Mutation: true, SecretResponse: true},
		{Parent: "api-key", Use: "list-metric", Method: http.MethodGet, Path: "/api/v1/apikeys/metric/list", Queries: withPaginationDefaults(queries("name", "page", "size")), SecretResponse: true},
		{Parent: "api-key", Use: "delete-metric", Method: http.MethodDelete, Path: "/api/v1/apikeys/metric/{key}", Mutation: true, Destructive: true, SecretPathParams: []string{"key"}},

		{Parent: "change-alarm", Use: "create", Method: http.MethodPost, Path: "/api/v1/alarms/change-method", Body: requiredObjectBody, Mutation: true},
		{Parent: "change-alarm", Use: "get", Method: http.MethodGet, Path: "/api/v1/alarms/change-method/{alarmId}", Queries: requiredQueries("end_time", "start_time")},
		{Parent: "change-alarm", Use: "delete-history", Method: http.MethodDelete, Path: "/api/v1/alarms/change-method/{alarmId}/histories", Mutation: true, Destructive: true},
		{Parent: "change-alarm", Use: "get-history", Method: http.MethodGet, Path: "/api/v1/alarms/change-method/{alarmId}/histories", Queries: append(requiredQueries("end_time", "start_time"), query("interval"))},
		{Parent: "change-alarm", Use: "delete", Method: http.MethodDelete, Path: "/api/v1/alarms/change-method/{alarm_id}", Mutation: true, Destructive: true},
		{Parent: "change-alarm", Use: "update", Method: http.MethodPut, Path: "/api/v1/alarms/change-method/{alarm_id}", Body: requiredObjectBody, Mutation: true},

		{Parent: "dashboard", Use: "list", Method: http.MethodGet, Path: "/api/v1/dashboards", Queries: queries("filter", "page", "searching-field", "searching-text", "size")},
		{Parent: "dashboard", Use: "create", Method: http.MethodPost, Path: "/api/v1/dashboards", Body: requiredObjectBody, Mutation: true},
		{Parent: "dashboard", Use: "update", Method: http.MethodPut, Path: "/api/v1/dashboards", Body: requiredObjectBody, Mutation: true},
		{Parent: "dashboard", Use: "clone", Method: http.MethodPost, Path: "/api/v1/dashboards/clone", Body: requiredObjectBody, Mutation: true},
		{Parent: "dashboard", Use: "create-default", Method: http.MethodPost, Path: "/api/v1/dashboards/default", Body: requiredObjectBody, Mutation: true},
		{Parent: "dashboard", Use: "create-default-infrastructure", Method: http.MethodPost, Path: "/api/v1/dashboards/default-infrastructure", Body: requiredObjectBody, Mutation: true},
		{Parent: "dashboard", Use: "favorite", Method: http.MethodPut, Path: "/api/v1/dashboards/favorite", Body: requiredObjectBody, Mutation: true},
		{Parent: "dashboard", Use: "get-by-name", Method: http.MethodGet, Path: "/api/v1/dashboards/name/{name}"},
		{Parent: "dashboard", Use: "delete-by-name", Method: http.MethodDelete, Path: "/api/v1/dashboards/name/{resourceId}", Mutation: true, Destructive: true},
		{Parent: "dashboard", Use: "rename", Method: http.MethodPut, Path: "/api/v1/dashboards/rename", Body: requiredObjectBody, Mutation: true},
		{Parent: "dashboard", Use: "delete", Method: http.MethodDelete, Path: "/api/v1/dashboards/{dashboard_id}", Mutation: true, Destructive: true},
		{Parent: "dashboard", Use: "get", Method: http.MethodGet, Path: "/api/v1/dashboards/{id}"},

		{Parent: "infrastructure", Use: "list-hosts", Method: http.MethodGet, Path: "/api/v1/infrastructure/hosts", Queries: withPaginationDefaults(queries("page", "searching_text", "size"))},
		{Parent: "infrastructure", Use: "delete-host", Method: http.MethodDelete, Path: "/api/v1/infrastructure/hosts/{id}", Mutation: true, Destructive: true},
		{Parent: "infrastructure", Use: "get-host", Method: http.MethodGet, Path: "/api/v1/infrastructure/hosts/{id}"},
		{Parent: "infrastructure", Use: "disable-host", Method: http.MethodPut, Path: "/api/v1/infrastructure/hosts/{id}/disabled", Mutation: true},
		{Parent: "infrastructure", Use: "enable-host", Method: http.MethodPut, Path: "/api/v1/infrastructure/hosts/{id}/enabled", Mutation: true},
		{Parent: "infrastructure", Use: "get-host-metric", Method: http.MethodGet, Path: "/api/v1/infrastructure/hosts/{id}/metric"},
		{Parent: "integration", Use: "install", Method: http.MethodPut, Path: "/api/v1/integrations/install/{id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "integration", Use: "list", Method: http.MethodGet, Path: "/api/v1/integrations/list", Queries: queries("page", "size")},
		{Parent: "integration", Use: "uninstall", Method: http.MethodPut, Path: "/api/v1/integrations/uninstall/{id}", Mutation: true, Destructive: true},
		{Parent: "integration", Use: "delete", Method: http.MethodDelete, Path: "/api/v1/integrations/{id}", Mutation: true, Destructive: true},
		{Parent: "integration", Use: "get", Method: http.MethodGet, Path: "/api/v1/integrations/{id}"},

		{Parent: "metric", Use: "get-dimensions", Method: http.MethodGet, Path: "/api/v1/metrics/dimensions", Queries: append(requiredQueries("name"), queries("dimensions", "end_time", "start_time")...)},
		{Parent: "metric", Use: "list-dimension-names", Method: http.MethodGet, Path: "/api/v1/metrics/dimensions-names"},
		{Parent: "metric", Use: "list-dimension-values", Method: http.MethodGet, Path: "/api/v1/metrics/dimensions-values", Queries: append(requiredQueries("dimension_name"), queries("dimensions", "end_time", "start_time")...)},
		{Parent: "metric", Use: "list-names", Method: http.MethodGet, Path: "/api/v1/metrics/metric-name", Queries: queries("end_time", "start_time")},
		{Parent: "metric-unit", Use: "list", Method: http.MethodGet, Path: "/api/v1/metricUnits/list", Queries: queries("page", "size")},
		{Parent: "metric-unit-mapping", Use: "list", Method: http.MethodGet, Path: "/api/v1/metric-unit-mappings/list", Queries: queries("isDefault", "name", "page", "size")},
		{Parent: "metric-unit-mapping-user", Use: "create", Method: http.MethodPost, Path: "/api/v1/metric-unit-mapping-users", Body: requiredObjectBody, Mutation: true},
		{Parent: "metric-unit-mapping-user", Use: "delete", Method: http.MethodDelete, Path: "/api/v1/metric-unit-mapping-users/{id}", Mutation: true, Destructive: true},
		{Parent: "statistic", Use: "get", Method: http.MethodGet, Path: "/api/v1/statistics", Queries: queries("alarm", "dimensions", "end_time", "group_by", "limit", "name", "period", "start_time", "statistics")},
		{Parent: "statistic", Use: "get-v2", Method: http.MethodPost, Path: "/api/v1/statistics", Body: requiredObjectBody},
		{Parent: "statistic", Use: "get-synthetic", Method: http.MethodGet, Path: "/api/v1/statistics/synthetics", Queries: queries("alarm", "dimensions", "end_time", "group_by", "name", "period", "start_time", "statistics")},

		{Parent: "variable", Use: "list", Method: http.MethodGet, Path: "/api/v1/dashboards/{dashboard_id}/variables"},
		{Parent: "variable", Use: "update", Method: http.MethodPut, Path: "/api/v1/dashboards/{dashboard_id}/variables", Body: requiredObjectBody, Mutation: true},
		{Parent: "variable", Use: "get", Method: http.MethodGet, Path: "/api/v1/dashboards/{dashboard_id}/variables/{variable_id}"},
		{Parent: "view", Use: "list", Method: http.MethodGet, Path: "/api/v1/dashboards/{dashboard_id}/views"},
		{Parent: "view", Use: "create", Method: http.MethodPost, Path: "/api/v1/dashboards/{dashboard_id}/views", Body: requiredObjectBody, Mutation: true},
		{Parent: "view", Use: "delete", Method: http.MethodDelete, Path: "/api/v1/dashboards/{dashboard_id}/views/{view_id}", Mutation: true, Destructive: true},
		{Parent: "view", Use: "get", Method: http.MethodGet, Path: "/api/v1/dashboards/{dashboard_id}/views/{view_id}"},
		{Parent: "view", Use: "update", Method: http.MethodPut, Path: "/api/v1/dashboards/{dashboard_id}/views/{view_id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "widget", Use: "update-layout", Method: http.MethodPut, Path: "/api/v1/dashboards/{dashboard_id}/widgets/layout/{widget_id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "widget", Use: "delete", Method: http.MethodDelete, Path: "/api/v1/dashboards/{dashboard_id}/widgets/{widget_id}", Mutation: true, Destructive: true},
		{Parent: "widget", Use: "get", Method: http.MethodGet, Path: "/api/v1/dashboards/{dashboard_id}/widgets/{widget_id}"},
		{Parent: "widget", Use: "update", Method: http.MethodPut, Path: "/api/v1/dashboards/{dashboard_id}/widgets/{widget_id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "widget-v2", Use: "create", Method: http.MethodPost, Path: "/api/v1/dashboards/{dashboard_id}/widgets/v2", Body: requiredObjectBody, Mutation: true},
		{Parent: "widget-v2", Use: "update", Method: http.MethodPut, Path: "/api/v1/dashboards/{dashboard_id}/widgets/v2/{widget_id}", Body: requiredObjectBody, Mutation: true},
	}
	for _, service := range []string{"vas", "vbackup", "vbandwidth", "vdb", "vlb", "vserver", "vstorage"} {
		operations = append(operations, infrastructureOperations(service)...)
	}
	return operations
}

func infrastructureOperations(service string) []operation {
	prefix := "/api/v1/infrastructure/" + service + "/hosts"
	return []operation{
		{Parent: "infrastructure", Use: "list-" + service + "-hosts", Method: http.MethodGet, Path: prefix, Queries: withPaginationDefaults(queries("name", "page", "size"))},
		{Parent: "infrastructure", Use: "delete-" + service + "-host", Method: http.MethodDelete, Path: prefix + "/{id}", Mutation: true, Destructive: true},
		{Parent: "infrastructure", Use: "update-" + service + "-host", Method: http.MethodPut, Path: prefix + "/{id}", Body: requiredObjectBody, Mutation: true},
		{Parent: "infrastructure", Use: "get-" + service + "-host-metric", Method: http.MethodGet, Path: prefix + "/{id}/metric"},
	}
}

var paginationDefaultValues = map[string]string{"page": "0", "size": "50"}

func withPaginationDefaults(parameters []queryParameter) []queryParameter {
	for i := range parameters {
		if def, ok := paginationDefaultValues[parameters[i].WireName]; ok {
			parameters[i].Default = def
		}
	}
	return parameters
}

func query(name string) queryParameter {
	return queryParameter{WireName: name, Flag: flagName(name), Usage: "Query parameter " + name}
}

func queries(names ...string) []queryParameter {
	parameters := make([]queryParameter, len(names))
	for i, name := range names {
		parameters[i] = query(name)
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
