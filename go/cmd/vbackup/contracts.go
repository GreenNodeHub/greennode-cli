package vbackup

import (
	"net/http"

	opengine "github.com/greennodehub/greennode-cli/internal/operation"
)

var (
	resourceID = pathParameter{Placeholder: "id", Flag: "id", Usage: "vBackup resource ID"}

	backendID             = queryParameter{WireName: "backendId", Flag: "backend-id", Usage: "Filter by backend ID", Kind: opengine.QueryID}
	projectID             = queryParameter{WireName: "projectId", Flag: "project-id", Usage: "Filter by project ID", Kind: opengine.QueryID}
	resourceIDQuery       = queryParameter{WireName: "id", Flag: "id", Usage: "Filter by resource ID", Kind: opengine.QueryID}
	name                  = queryParameter{WireName: "name", Flag: "name", Usage: "Filter by name"}
	serverID              = queryParameter{WireName: "serverId", Flag: "server-id", Usage: "Filter by vServer ID", Kind: opengine.QueryID}
	backupInstanceID      = queryParameter{WireName: "backupInstanceId", Flag: "backup-instance-id", Usage: "Filter by backup server ID", Kind: opengine.QueryID}
	backupInstancePointID = queryParameter{WireName: "backupInstancePointId", Flag: "backup-instance-point-id", Usage: "Filter by backup server point ID", Kind: opengine.QueryID}
	backupVolumePointID   = queryParameter{WireName: "backupVolumePointId", Flag: "backup-volume-point-id", Usage: "Filter by backup volume point ID", Kind: opengine.QueryID}
	volumeID              = queryParameter{WireName: "volumeId", Flag: "volume-id", Usage: "Filter by volume ID", Kind: opengine.QueryID}
	backend               = queryParameter{WireName: "backend", Flag: "backend", Usage: "Filter by backend name"}
	policyType            = queryParameter{WireName: "type", Flag: "type", Usage: "Filter by destination type"}
	page                  = queryParameter{WireName: "page", Flag: "page", Usage: "Page number (starts at 1)", Kind: opengine.QueryInteger, Minimum: 1}
	size                  = queryParameter{WireName: "size", Flag: "size", Usage: "Maximum items per page", Kind: opengine.QueryInteger}
)

func allOperations() []operation {
	return []operation{
		{Parent: "backend", Use: "list", Short: "List vBackup backends", Method: http.MethodGet, Path: "/v1/backends", Queries: []queryParameter{backend, page, size}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "vserver", Use: "get-instance-point", Short: "Get a vServer backup server point", Method: http.MethodGet, Path: "/v1/vserver/backup-instance-points/{id}", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "vserver", Use: "list-volume-points", Short: "List vServer backup volume points", Method: http.MethodGet, Path: "/v1/vserver/backup-instance-points/{id}/backup-volume-points", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true, Extra: responseArray},
		{Parent: "vserver", Use: "list-instances", Short: "List backup servers for vServer", Method: http.MethodGet, Path: "/v1/vserver/backup-instances", Queries: []queryParameter{backendID, projectID}, Status: http.StatusOK, ResponseBody: true, Extra: responseArray},
		{Parent: "vserver", Use: "create-instance", Short: "Create backup servers from vServers", Method: http.MethodPost, Path: "/v1/vserver/backup-instances", Body: requiredObjectBody, Mutation: true, Status: http.StatusOK, ResponseBody: true, Extra: responseArray},
		{Parent: "vserver", Use: "get-instance", Short: "Get a vServer backup server", Method: http.MethodGet, Path: "/v1/vserver/backup-instances/{id}", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "vserver", Use: "list-instance-points", Short: "List points for a vServer backup server", Method: http.MethodGet, Path: "/v1/vserver/backup-instances/{id}/backup-instance-points", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true, Extra: responseArray},
		{Parent: "vserver", Use: "get-volume-point", Short: "Get a vServer backup volume point", Method: http.MethodGet, Path: "/v1/vserver/backup-volume-points/{id}", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "destination", Use: "list", Short: "List backup destinations", Method: http.MethodGet, Path: "/v1/backup-destinations", Queries: []queryParameter{backendID, projectID, name, policyType, page, size}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "policy", Use: "list", Short: "List backup policies", Method: http.MethodGet, Path: "/v1/backup-policies", Queries: []queryParameter{backendID, projectID, name, page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "policy", Use: "create", Short: "Create a backup policy", Method: http.MethodPost, Path: "/v1/backup-policies", Body: requiredObjectBody, Mutation: true, Status: http.StatusCreated, ResponseBody: true},
		{Parent: "policy", Use: "delete", Short: "Delete a backup policy", Method: http.MethodDelete, Path: "/v1/backup-policies/{id}", Paths: []pathParameter{resourceID}, Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusNoContent}, Status: http.StatusNoContent, ResponseBody: false},
		{Parent: "policy", Use: "get", Short: "Get a backup policy", Method: http.MethodGet, Path: "/v1/backup-policies/{id}", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "policy", Use: "update", Short: "Update a backup policy", Method: http.MethodPut, Path: "/v1/backup-policies/{id}", Paths: []pathParameter{resourceID}, Body: requiredObjectBody, Mutation: true, Status: http.StatusOK, ResponseBody: true},

		{Parent: "server", Use: "list", Short: "List backup servers", Method: http.MethodGet, Path: "/v1/backup-instances", Queries: []queryParameter{backendID, projectID, resourceIDQuery, name, serverID, page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "server", Use: "create", Short: "Create a backup server", Method: http.MethodPost, Path: "/v1/backup-instances", Body: requiredObjectBody, Mutation: true, Status: http.StatusCreated, ResponseBody: true, Extra: responseArray},
		{Parent: "server", Use: "list-protected", Short: "List protected vServers", Method: http.MethodGet, Path: "/v1/backup-instances/protected-servers", Queries: []queryParameter{backendID, projectID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "server", Use: "delete", Short: "Delete a backup server", Method: http.MethodDelete, Path: "/v1/backup-instances/{id}", Paths: []pathParameter{resourceID}, Mutation: true, Destructive: true, EmptySuccess: []int{http.StatusNoContent}, Status: http.StatusNoContent, ResponseBody: false},
		{Parent: "server", Use: "get", Short: "Get a backup server", Method: http.MethodGet, Path: "/v1/backup-instances/{id}", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "server", Use: "list-points", Short: "List points for a backup server", Method: http.MethodGet, Path: "/v1/backup-instances/{id}/backup-instance-points", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true, Extra: responseArray},
		{Parent: "server", Use: "disable", Short: "Disable a backup server", Method: http.MethodPut, Path: "/v1/backup-instances/{id}/disabled", Paths: []pathParameter{resourceID}, Mutation: true, EmptySuccess: []int{http.StatusOK}, Status: http.StatusOK, ResponseBody: false},
		{Parent: "server", Use: "enable", Short: "Enable a backup server", Method: http.MethodPut, Path: "/v1/backup-instances/{id}/enabled", Paths: []pathParameter{resourceID}, Mutation: true, EmptySuccess: []int{http.StatusOK}, Status: http.StatusOK, ResponseBody: false},
		{Parent: "server", Use: "update-policy", Short: "Update a backup server policy", Method: http.MethodPut, Path: "/v1/backup-instances/{id}/policies", Paths: []pathParameter{resourceID}, Body: requiredObjectBody, Mutation: true, EmptySuccess: []int{http.StatusOK}, Status: http.StatusOK, ResponseBody: false},
		{Parent: "server", Use: "list-volumes", Short: "List volumes of a backup server", Method: http.MethodGet, Path: "/v1/backup-instances/{id}/volumes", Paths: []pathParameter{resourceID}, Status: http.StatusOK, ResponseBody: true, Extra: responseArray},
		{Parent: "server", Use: "update-volume", Short: "Update backup status for a volume", Method: http.MethodPut, Path: "/v1/backup-instances/{id}/volumes", Paths: []pathParameter{resourceID}, Body: requiredObjectBody, Mutation: true, EmptySuccess: []int{http.StatusOK}, Status: http.StatusOK, ResponseBody: false},

		{Parent: "configuration", Use: "get", Short: "Get vBackup configuration", Method: http.MethodGet, Path: "/v1/configurations", Status: http.StatusOK, ResponseBody: true},

		{Parent: "history", Use: "list-backups", Short: "List backup server history", Method: http.MethodGet, Path: "/v1/histories/backup-instances", Queries: []queryParameter{backendID, projectID, resourceIDQuery, serverID, backupInstanceID, page, size}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "history", Use: "list-restorations", Short: "List restoration history", Method: http.MethodGet, Path: "/v1/histories/restoration", Queries: []queryParameter{backendID, projectID, resourceIDQuery, serverID, backupInstanceID, backupInstancePointID, volumeID, backupVolumePointID, page, size}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "volume", Use: "usage", Short: "List backup usage for volumes", Method: http.MethodPost, Path: "/v1/volume-usage", Body: requiredObjectBody, Status: http.StatusOK, ResponseBody: true, Extra: responseArray},
	}
}
