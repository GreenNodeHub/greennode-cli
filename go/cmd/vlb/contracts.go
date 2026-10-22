package vlb

import (
	"fmt"
	"net/http"

	opengine "github.com/greennodehub/greennode-cli/internal/operation"
)

var (
	projectID      = pathParameter{Placeholder: "projectId", Flag: "project-id", Usage: "vLB project ID"}
	certificateID  = pathParameter{Placeholder: "caId", Flag: "certificate-id", Usage: "Certificate ID"}
	loadBalancerID = pathParameter{Placeholder: "loadBalancerId", Flag: "load-balancer-id", Usage: "Load balancer ID"}
	listenerID     = pathParameter{Placeholder: "listenerId", Flag: "listener-id", Usage: "Listener ID"}
	l7PolicyID     = pathParameter{Placeholder: "l7PolicyId", Flag: "l7-policy-id", Usage: "L7 policy ID"}
	policyID       = pathParameter{Placeholder: "policyId", Flag: "l7-policy-id", Usage: "L7 policy ID"}
	poolID         = pathParameter{Placeholder: "poolId", Flag: "pool-id", Usage: "Pool ID"}
	subnetID       = pathParameter{Placeholder: "subnetId", Flag: "subnet-id", Usage: "Subnet ID"}
)

var (
	certificateListQueries = []queryParameter{
		{WireName: "name", Flag: "name", Usage: "Filter certificates by name"},
		{WireName: "page", Flag: "page", Usage: "Page number (the API default is 1)", Kind: queryInteger},
		{WireName: "size", Flag: "size", Usage: "Number of certificates per page (the API default is 10)", Kind: queryInteger},
	}
	loadBalancerListQueries = []queryParameter{
		{WireName: "filter", Flag: "filter", Usage: "Required FilterRequest query object as JSON", Required: true, Kind: queryObject},
		{WireName: "pageRequest", Flag: "page-request", Usage: "Required PageRequest query object as JSON", Required: true, Kind: queryObject},
	}
)

func objectBody(schema string, requiredFields ...string) *bodyContract {
	return &bodyContract{
		Kind:           opengine.ObjectBody,
		Style:          opengine.BodyStyleJSONObject,
		Usage:          fmt.Sprintf("Request body as a %s JSON object", schema),
		Name:           schema,
		RequiredFields: requiredFields,
	}
}

func allOperations() []operation {
	return []operation{
		{Parent: "certificate", Use: "list", Short: "List vLB certificates", OperationID: "getAllListCertificateAuthority", Method: http.MethodGet, Path: "/v2/{projectId}/cas", Paths: []pathParameter{projectID}, Queries: certificateListQueries, Status: http.StatusOK, ResponseBody: true},
		{Parent: "certificate", Use: "import", Short: "Import a vLB certificate", OperationID: "importCA", Method: http.MethodPost, Path: "/v2/{projectId}/cas", Paths: []pathParameter{projectID}, Body: objectBody("ImportCARequest", "name", "type"), Status: http.StatusCreated, ResponseBody: true},
		{Parent: "certificate", Use: "delete", Short: "Delete a vLB certificate", OperationID: "deleteCertificateAuthority", Method: http.MethodDelete, Path: "/v2/{projectId}/cas/{caId}", Paths: []pathParameter{projectID, certificateID}, Status: http.StatusNoContent, Destructive: true},
		{Parent: "certificate", Use: "get", Short: "Get a vLB certificate", OperationID: "getCertificateAuthorityById", Method: http.MethodGet, Path: "/v2/{projectId}/cas/{caId}", Paths: []pathParameter{projectID, certificateID}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "load-balancer", Use: "list", Short: "List vLB load balancers", OperationID: "getListLoadBalancer", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers", Paths: []pathParameter{projectID}, Queries: loadBalancerListQueries, Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "create", Short: "Create a vLB load balancer", OperationID: "createLoadBalancer", Method: http.MethodPost, Path: "/v2/{projectId}/loadBalancers", Paths: []pathParameter{projectID}, Body: objectBody("CreateLoadBalancerRequest", "name", "packageId", "scheme", "subnetId", "type"), Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "list-all-packages", Short: "List all vLB packages", OperationID: "getAllPackages", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/all_packages", Paths: []pathParameter{projectID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "list-headers", Short: "List supported vLB headers", OperationID: "getHeaders", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/headers", Paths: []pathParameter{projectID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "list-packages", Short: "List active vLB packages", OperationID: "getPackagesActive", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/packages", Paths: []pathParameter{projectID}, Queries: []queryParameter{{WireName: "zoneId", Flag: "zone-id", Usage: "Filter packages by zone ID", Kind: queryID}}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "list-by-subnet", Short: "List load balancers attached to a subnet", OperationID: "getLoadBalancerBySubnetId", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/subnet/{subnetId}", Paths: []pathParameter{projectID, subnetID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "delete", Short: "Delete a vLB load balancer", OperationID: "deleteLoadBalancer", Method: http.MethodDelete, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}", Paths: []pathParameter{projectID, loadBalancerID}, Status: http.StatusAccepted, Destructive: true},
		{Parent: "load-balancer", Use: "get", Short: "Get a vLB load balancer", OperationID: "getLoadBalancer", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}", Paths: []pathParameter{projectID, loadBalancerID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "update-bandwidth", Short: "Update a load balancer bandwidth package", OperationID: "updateLoadBalancerBandwidth", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/bandwidth", Paths: []pathParameter{projectID, loadBalancerID}, Body: objectBody("UpdateBandwidthPackageRequest", "bandWidthPackageId"), Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "clone", Short: "Clone a vLB load balancer", OperationID: "cloneLoadBalancer", Method: http.MethodPost, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/clone", Paths: []pathParameter{projectID, loadBalancerID}, Body: objectBody("CloneLoadBalancerRequest", "name", "packageId"), Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "get-clone-metadata", Short: "Get metadata for cloning a load balancer", OperationID: "getCloneInfo", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/cloneMetadata", Paths: []pathParameter{projectID, loadBalancerID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "update-rebalancing", Short: "Update load balancer HA rebalancing", OperationID: "haConfiguration", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/rebalancing", Paths: []pathParameter{projectID, loadBalancerID}, Body: objectBody("ScaleConfigRequest", "networking", "scaling"), Status: http.StatusAccepted},
		{Parent: "load-balancer", Use: "resize", Short: "Resize a vLB load balancer", OperationID: "resizeLoadBalancer", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/resize", Paths: []pathParameter{projectID, loadBalancerID}, Body: objectBody("ResizeLoadBalancerRequest", "packageId"), Status: http.StatusOK, ResponseBody: true},
		{Parent: "load-balancer", Use: "list-scale-history", Short: "List scaling history for a load balancer", OperationID: "listByLoadBalancer", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/scale-history", Paths: []pathParameter{projectID, loadBalancerID}, Status: http.StatusOK, ResponseBody: true},

		{Parent: "listener", Use: "list", Short: "List listeners for a load balancer", OperationID: "getListListenersByLoadBalancer", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners", Paths: []pathParameter{projectID, loadBalancerID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "listener", Use: "create", Short: "Create a vLB listener", OperationID: "createListener", Method: http.MethodPost, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners", Paths: []pathParameter{projectID, loadBalancerID}, Body: objectBody("CreateListenerRequest", "listenerName", "listenerProtocol", "listenerProtocolPort", "timeoutClient", "timeoutConnection", "timeoutMember"), Status: http.StatusOK, ResponseBody: true},
		{Parent: "listener", Use: "delete", Short: "Delete a vLB listener", OperationID: "deleteListener", Method: http.MethodDelete, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}", Paths: []pathParameter{projectID, loadBalancerID, listenerID}, Status: http.StatusAccepted, Destructive: true},
		{Parent: "listener", Use: "get", Short: "Get a vLB listener", OperationID: "getListener", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}", Paths: []pathParameter{projectID, loadBalancerID, listenerID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "listener", Use: "update", Short: "Update a vLB listener", OperationID: "updateListener", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}", Paths: []pathParameter{projectID, loadBalancerID, listenerID}, Body: objectBody("UpdateListenerRequest", "timeoutClient", "timeoutConnection", "timeoutMember"), Status: http.StatusOK},

		{Parent: "l7-policy", Use: "list", Short: "List L7 policies for a listener", OperationID: "getListL7Policies", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/l7policies", Paths: []pathParameter{projectID, loadBalancerID, listenerID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "l7-policy", Use: "create", Short: "Create a listener L7 policy", OperationID: "createPolicy", Method: http.MethodPost, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/l7policies", Paths: []pathParameter{projectID, loadBalancerID, listenerID}, Body: objectBody("CreateL7PolicyRequest", "action", "name"), Status: http.StatusOK, ResponseBody: true},
		{Parent: "l7-policy", Use: "get", Short: "Get a listener L7 policy", OperationID: "getL7Policy", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/l7policies/{l7PolicyId}", Paths: []pathParameter{projectID, loadBalancerID, listenerID, l7PolicyID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "l7-policy", Use: "delete", Short: "Delete a listener L7 policy", OperationID: "deletePolicy", Method: http.MethodDelete, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/l7policies/{policyId}", Paths: []pathParameter{projectID, loadBalancerID, listenerID, policyID}, Status: http.StatusAccepted, Destructive: true},
		{Parent: "l7-policy", Use: "update", Short: "Update a listener L7 policy", OperationID: "updateL7Policy", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/l7policies/{policyId}", Paths: []pathParameter{projectID, loadBalancerID, listenerID, policyID}, Body: objectBody("UpdateL7PolicyRequest", "action"), Status: http.StatusOK},
		{Parent: "l7-policy", Use: "reorder", Short: "Reorder L7 policies for a listener", OperationID: "reorderPolicies", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/listeners/{listenerId}/reorderL7Policies", Paths: []pathParameter{projectID, loadBalancerID, listenerID}, Body: objectBody("ReorderPoliciesRequest", "policies"), Status: http.StatusOK},

		{Parent: "pool", Use: "list", Short: "List pools for a load balancer", OperationID: "getListPools", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools", Paths: []pathParameter{projectID, loadBalancerID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "pool", Use: "create", Short: "Create a vLB pool", OperationID: "createPool", Method: http.MethodPost, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools", Paths: []pathParameter{projectID, loadBalancerID}, Body: objectBody("CreatePoolRequest", "algorithm", "healthMonitor", "poolName", "poolProtocol"), Status: http.StatusOK, ResponseBody: true},
		{Parent: "pool", Use: "delete", Short: "Delete a vLB pool", OperationID: "deletePool", Method: http.MethodDelete, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}", Paths: []pathParameter{projectID, loadBalancerID, poolID}, Status: http.StatusAccepted, Destructive: true},
		{Parent: "pool", Use: "get", Short: "Get a vLB pool", OperationID: "getPool", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}", Paths: []pathParameter{projectID, loadBalancerID, poolID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "pool", Use: "update", Short: "Update a vLB pool", OperationID: "updatePool", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}", Paths: []pathParameter{projectID, loadBalancerID, poolID}, Body: objectBody("UpdatePoolRequest", "algorithm"), Status: http.StatusOK},
		{Parent: "pool", Use: "get-health-monitor", Short: "Get a pool health monitor", OperationID: "getHealthMonitorFromPool", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}/healthMonitor", Paths: []pathParameter{projectID, loadBalancerID, poolID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "pool", Use: "list-members", Short: "List members of a vLB pool", OperationID: "getMembersFromPool", Method: http.MethodGet, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}/members", Paths: []pathParameter{projectID, loadBalancerID, poolID}, Status: http.StatusOK, ResponseBody: true},
		{Parent: "pool", Use: "update-members", Short: "Replace the members of a vLB pool", OperationID: "changesMembers", Method: http.MethodPut, Path: "/v2/{projectId}/loadBalancers/{loadBalancerId}/pools/{poolId}/members", Paths: []pathParameter{projectID, loadBalancerID, poolID}, Body: objectBody("ChangeMembersRequest", "members"), Status: http.StatusOK},
	}
}
