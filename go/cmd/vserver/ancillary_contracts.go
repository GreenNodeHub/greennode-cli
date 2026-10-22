package vserver

import "net/http"

var (
	ancillaryOptionalNamePageSize = []ancillaryQueryParameter{
		{Name: "name", Kind: ancillaryQueryString},
		{Name: "page", Kind: ancillaryQueryInteger},
		{Name: "size", Kind: ancillaryQueryInteger},
	}
	ancillaryRequiredNamePageSize = []ancillaryQueryParameter{
		{Name: "name", Required: true, Kind: ancillaryQueryString},
		{Name: "page", Required: true, Kind: ancillaryQueryString},
		{Name: "size", Required: true, Kind: ancillaryQueryString},
	}
	ancillaryHistoryQueries = []ancillaryQueryParameter{
		{Name: "page", Kind: ancillaryQueryInteger},
		{Name: "serverId", Kind: ancillaryQueryString},
		{Name: "size", Kind: ancillaryQueryInteger},
		{Name: "status", Kind: ancillaryQueryString},
	}
)

func ancillaryOperations() []ancillaryOperation {
	operations := []ancillaryOperation{
		ancillary("floating-ip", "list-elastic", "Floating-IP", "listElasticIpUsingGET", http.MethodGet, "/v2/{projectId}/elastic-ips", 200, true, ancillaryOptionalNamePageSize, nil),
		ancillaryWithProjectScope(ancillary("history", "list-server-migrations", "History", "listServerMigrationHistoryUsingGET", http.MethodGet, "/v1/{project_id}/histories/server-migration", 200, true, ancillaryHistoryQueries, nil), "project_id"),

		ancillary("interconnect", "list", "Interconnect", "listUsingGET", http.MethodGet, "/v2/{projectId}/interconnects", 200, true, ancillaryOptionalNamePageSize, nil),
		ancillary("interconnect", "create", "Interconnect", "createUsingPOST_1", http.MethodPost, "/v2/{projectId}/interconnects", 200, true, nil, ancillaryBody("CreateInterconnectRequest", "name", "packageId", "typeId")),
		ancillary("interconnect", "list-circuit-types", "Interconnect", "getTypesUsingGET", http.MethodGet, "/v2/{projectId}/interconnects/circuit-types", 200, true, nil, nil),
		ancillary("interconnect", "list-packages", "Interconnect", "listPackagesUsingGET", http.MethodGet, "/v2/{projectId}/interconnects/packages", 200, true, nil, nil),
		ancillary("interconnect", "delete", "Interconnect", "deleteUsingDELETE_1", http.MethodDelete, "/v2/{projectId}/interconnects/{interconnectId}", 204, false, nil, nil),
		ancillary("interconnect", "get", "Interconnect", "getUsingGET_1", http.MethodGet, "/v2/{projectId}/interconnects/{interconnectId}", 200, true, nil, nil),
		ancillary("interconnect", "update", "Interconnect", "updateUsingPUT", http.MethodPut, "/v2/{projectId}/interconnects/{interconnectId}", 200, true, nil, ancillaryBody("UpdateInterconnectRequest")),
		ancillary("interconnect", "change-package", "Interconnect", "changePackageUsingPUT", http.MethodPut, "/v2/{projectId}/interconnects/{interconnectId}/change-package", 200, true, nil, ancillaryBody("ChangePackageInterconnectRequest", "packageId")),
		ancillary("interconnect", "list-connections", "Interconnect", "listConnectionUsingGET", http.MethodGet, "/v2/{projectId}/interconnects/{interconnectId}/connections", 200, true, ancillaryOptionalNamePageSize, nil),
		ancillary("interconnect", "create-connection", "Interconnect", "createConnectionUsingPOST", http.MethodPost, "/v2/{projectId}/interconnects/{interconnectId}/connections", 200, true, nil, ancillaryBody("CreateInterconnectionRequest", "name", "networkId", "subnets")),
		ancillary("interconnect", "delete-connection", "Interconnect", "deleteConnectionUsingDELETE", http.MethodDelete, "/v2/{projectId}/interconnects/{interconnectId}/connections/{interconnectionId}", 204, false, nil, nil),
		ancillary("interconnect", "get-connection", "Interconnect", "getConnectionUsingGET", http.MethodGet, "/v2/{projectId}/interconnects/{interconnectId}/connections/{interconnectionId}", 200, true, nil, nil),
		ancillary("interconnect", "update-connection-subnets", "Interconnect", "updateSubnetsUsingPUT", http.MethodPut, "/v2/{projectId}/interconnects/{interconnectId}/connections/{interconnectionId}", 200, true, nil, ancillaryBody("UpdateSubnetsInterconnectionRequest", "subnets")),
		ancillary("interconnect", "ping", "Interconnect", "pingUsingPUT", http.MethodPut, "/v2/{projectId}/interconnects/{interconnectId}/ping", 200, true, nil, ancillaryBody("PingInterconnectRequest")),

		ancillary("interface-peering", "list", "Interface-peering", "listPeeringWithPagingUsingGET", http.MethodGet, "/v2/{projectId}/peering", 200, true, ancillaryRequiredNamePageSize, nil),
		ancillary("interface-peering", "delete", "Interface-peering", "deletePeeringUsingDELETE", http.MethodDelete, "/v2/{projectId}/peering/{peeringId}", 200, false, nil, nil),

		ancillary("marketplace", "list-app-categories", "Market-place", "listAppCategoryUsingGET", http.MethodGet, "/v1/app-category", 200, true, nil, nil),
		ancillary("marketplace", "get-app-instance", "Market-place", "getAppInstanceUsingGET", http.MethodGet, "/v1/app-instance/{appInstanceUuid}", 200, true, nil, nil),
		ancillary("marketplace", "get-app-package", "Market-place", "getAppPackageDetailUsingGET", http.MethodGet, "/v1/app-package/{appCategoryId}/{appPackageId}/{appVersionUuid}", 200, true, nil, nil),
		ancillaryWithPathKind(ancillary("marketplace", "get-app-template", "Market-place", "getAppTemplateUsingGET", http.MethodGet, "/v1/app-template/{appTemplateId}", 200, true, nil, nil), "appTemplateId", ancillaryPathInt32),
		ancillary("marketplace-migration", "migrate", "Market-place-migration", "migrateMpUsingPOST", http.MethodPost, "/v1/mp-migrate", 200, true, []ancillaryQueryParameter{{Name: "all", Required: true, Kind: ancillaryQueryBoolean}}, nil),

		ancillary("network-acl", "create", "Network-acl", "createNetworkAclUsingPOST", http.MethodPost, "/v2/{projectId}/network-acl", 201, true, nil, ancillaryBody("CreateNetworkAclRequest", "name", "vpc")),
		ancillary("network-acl", "list", "Network-acl", "getNetworkAclsUsingGET", http.MethodGet, "/v2/{projectId}/network-acl/list", 200, true, ancillaryRequiredNamePageSize, nil),
		ancillary("network-acl", "update-rules", "Network-acl", "updateRulesUsingPUT", http.MethodPut, "/v2/{projectId}/network-acl/{aclId}/rules", 200, true, nil, ancillaryBody("UpdateNetworkAclRulesRequest")),
		ancillary("network-acl", "delete", "Network-acl", "deleteNetworkAclUsingDELETE", http.MethodDelete, "/v2/{projectId}/network-acl/{networkAclUuid}", 204, false, nil, nil),
		ancillary("network-acl", "get", "Network-acl", "getNetworkAclUsingGET", http.MethodGet, "/v2/{projectId}/network-acl/{uuid}", 200, true, nil, nil),
		ancillary("network-acl", "list-rules", "Network-acl", "getNetworkAclPolicyRulesUsingGET", http.MethodGet, "/v2/{projectId}/network-acl/{uuid}/rules", 200, true, nil, nil),
		ancillary("network-acl", "update-subnets", "Network-acl", "updateAssociatedSubnetsUsingPUT", http.MethodPut, "/v2/{projectId}/network-acl/{uuid}/subnets", 200, true, nil, ancillaryBody("UpdateNetworkAclSubnetsRequest")),

		ancillary("persistent-volume", "list", "Persistent-volume", "listPersistentVolumeUsingGET", http.MethodGet, "/v2/{projectId}/persistent-volumes", 200, true, ancillaryOptionalNamePageSize, nil),
		ancillary("persistent-volume", "delete", "Persistent-volume", "deletePersistentVolumeUsingDELETE", http.MethodDelete, "/v2/{projectId}/persistent-volumes/{pvId}", 200, true, nil, ancillaryBody("DeletePersistentVolumeBackendRequest")),
		ancillary("project", "list", "Project", "listProjectUsingGET", http.MethodGet, "/v1/projects", 200, true, nil, nil),
		ancillary("project", "get", "Project", "getProjectUsingGET", http.MethodGet, "/v1/projects/{project_id}", 200, true, nil, nil),
		ancillary("protocol", "list", "Protocol", "listProtocolUsingGET", http.MethodGet, "/v2/protocols", 200, true, nil, nil),
		ancillary("quota", "get-used", "Quota", "listQuotaUsedUsingGET", http.MethodGet, "/v2/{projectId}/quotas/quotaUsed", 200, true, nil, nil),
		ancillary("region", "list", "Region", "getALlRegionUsingGET", http.MethodGet, "/v2/{projectId}/region", 200, true, nil, nil),
		ancillary("region", "validate-user", "Region", "listQuotaUsedUsingGET_1", http.MethodGet, "/v2/{projectId}/region/{regionId}/users/validation", 200, true, nil, nil),

		ancillary("route-table", "list", "Route-table", "listRouteTableUsingGET", http.MethodGet, "/v2/{projectId}/route-table", 200, true, ancillaryRequiredNamePageSize, nil),
		ancillary("route-table", "create", "Route-table", "createRouteTableUsingPOST", http.MethodPost, "/v2/{projectId}/route-table", 200, true, nil, ancillaryBody("CreateRouteTableRequest", "name", "networkId")),
		ancillary("route-table", "list-routes", "Route-table", "listRouteFromRouteTableIdUsingGET", http.MethodGet, "/v2/{projectId}/route-table/route/{routeTableId}", 200, true, nil, nil),
		ancillary("route-table", "delete", "Route-table", "deleteRouteTableUsingDELETE", http.MethodDelete, "/v2/{projectId}/route-table/{routeId}", 202, false, nil, nil),
		ancillary("route-table", "get", "Route-table", "getRouteTableUsingGET", http.MethodGet, "/v2/{projectId}/route-table/{uuid}", 200, true, nil, nil),
		ancillary("route-table", "update-routes", "Route-table", "updateRouteTableDetailUsingPUT", http.MethodPut, "/v2/{projectId}/route-table/{uuid}/routes", 200, false, nil, ancillaryBody("ChangeRoutesRequest")),

		ancillary("tag", "list", "Tag", "listTagUsingGET", http.MethodGet, "/v2/{projectId}/tag", 200, true, ancillaryOptionalNamePageSize, nil),
		ancillary("tag", "get-quota", "Tag", "getTagQuotaUsingGET", http.MethodGet, "/v2/{projectId}/tag/quota", 200, true, nil, nil),
		ancillary("tag", "list-resource-tags", "Tag", "listTagByResourceIdUsingGET", http.MethodGet, "/v2/{projectId}/tag/resource/{resourceId}", 200, true, nil, nil),
		ancillary("tag", "list-resources", "Tag", "getResourceByTagUsingGET", http.MethodGet, "/v2/{projectId}/tag/{tagId}/resource-types/{resource-types}", 200, true, nil, nil),

		ancillary("virtual-ip-address", "create-public", "Virtual-ip-address", "createPublicVirtualIpAddressUsingPOST", http.MethodPost, "/v2/{projectId}/public-vips", 200, true, nil, ancillaryBody("CreatePublicVipRequest", "name", "type")),
		ancillary("virtual-ip-address", "list-public-interfaces", "Virtual-ip-address", "listExternalNetworkInterfaceUsingGET", http.MethodGet, "/v2/{projectId}/public-vips/externalNetworkInterfaces", 200, true, nil, nil),
		ancillary("virtual-ip-address", "delete-public", "Virtual-ip-address", "deletePublicVirtualIpAddressUsingDELETE", http.MethodDelete, "/v2/{projectId}/public-vips/{publicVipId}", 204, false, nil, nil),
		ancillary("virtual-ip-address", "create-public-address-pair", "Virtual-ip-address", "createPublicAddressPairUsingPOST", http.MethodPost, "/v2/{projectId}/public-vips/{virtualIpAddressId}/addressPairs", 201, true, nil, ancillaryBody("CreatePublicAddressPairRequest", "networkInterfaceId")),
		ancillary("virtual-ip-address", "delete-public-address-pair", "Virtual-ip-address", "deletePublicAddressPairUsingDELETE", http.MethodDelete, "/v2/{projectId}/public-vips/{virtualIpAddressId}/addressPairs/{addressPairId}", 204, false, nil, nil),
		ancillary("virtual-ip-address", "list-private", "Virtual-ip-address", "getListVirtualIpAddressWithPagingUsingGET", http.MethodGet, "/v2/{projectId}/virtualIpAddress", 200, true, ancillaryRequiredNamePageSize, nil),
		ancillary("virtual-ip-address", "create-private", "Virtual-ip-address", "createPrivateVirtualIpAddressUsingPOST", http.MethodPost, "/v2/{projectId}/virtualIpAddress", 201, true, nil, ancillaryBody("CreateVirtualIpAddressRequest", "mode", "name", "subnetId")),
		ancillary("virtual-ip-address", "list-private-interfaces", "Virtual-ip-address", "listInternalNetworkInterfaceUsingGET", http.MethodGet, "/v2/{projectId}/virtualIpAddress/internalNetworkInterfaces", 200, true, []ancillaryQueryParameter{{Name: "subnetId", Kind: ancillaryQueryString}, {Name: "zoneId", Kind: ancillaryQueryString}}, nil),
		ancillary("virtual-ip-address", "get-private", "Virtual-ip-address", "getVirtualIpAddressUsingGET", http.MethodGet, "/v2/{projectId}/virtualIpAddress/{vipId}", 200, true, nil, nil),
		ancillary("virtual-ip-address", "delete-private", "Virtual-ip-address", "deletePrivateVirtualIpAddressUsingDELETE", http.MethodDelete, "/v2/{projectId}/virtualIpAddress/{virtualIpAddressId}", 204, false, nil, nil),
		ancillary("virtual-ip-address", "update-private", "Virtual-ip-address", "updateVirtualIpAddressUsingPUT", http.MethodPut, "/v2/{projectId}/virtualIpAddress/{virtualIpAddressId}", 200, true, nil, ancillaryBody("UpdateVirtualIpAddressRequest", "mode")),
		ancillary("virtual-ip-address", "list-private-address-pairs", "Virtual-ip-address", "getListAddressPairUsingGET", http.MethodGet, "/v2/{projectId}/virtualIpAddress/{virtualIpAddressId}/addressPairs", 200, true, nil, nil),
		ancillary("virtual-ip-address", "create-private-address-pair", "Virtual-ip-address", "createPrivateAddressPairUsingPOST", http.MethodPost, "/v2/{projectId}/virtualIpAddress/{virtualIpAddressId}/addressPairs", 201, true, nil, ancillaryBody("CreatePrivateAddressPairRequest", "internalNetworkInterfaceId")),
		ancillary("virtual-ip-address", "delete-private-address-pair", "Virtual-ip-address", "deletePrivateAddressPairUsingDELETE", http.MethodDelete, "/v2/{projectId}/virtualIpAddress/{virtualIpAddressId}/addressPairs/{addressPairId}", 204, false, nil, nil),
		ancillary("virtual-ip-address", "get-private-address-pair", "Virtual-ip-address", "getSpecificAddressPairUsingGET", http.MethodGet, "/v2/{projectId}/virtualIpAddress/{virtualIpAddressId}/addressPairs/{addressPairId}", 200, true, nil, nil),

		ancillary("virtual-subnet", "delete-address-pair", "Virtual-subnet", "deleteAddressPairUsingDELETE", http.MethodDelete, "/v2/{projectId}/virtual-subnets/addressPairs/{addressPairId}", 204, false, nil, nil),
		ancillary("virtual-subnet", "list-address-pairs", "Virtual-subnet", "getListAddressPairUsingGET_1", http.MethodGet, "/v2/{projectId}/virtual-subnets/{secondarySubnetId}/addressPairs", 200, true, nil, nil),
		ancillary("virtual-subnet", "create-address-pair", "Virtual-subnet", "addAddressPairUsingPOST", http.MethodPost, "/v2/{projectId}/virtual-subnets/{secondarySubnetId}/addressPairs", 201, true, nil, ancillaryBody("CreatePrivateAddressPairRequest", "internalNetworkInterfaceId")),
		ancillary("zone", "list", "Zone", "listZoneUsingGET", http.MethodGet, "/v1/{projectId}/zones", 200, true, nil, nil),
		ancillary("zone", "get", "Zone", "getZoneUsingGET", http.MethodGet, "/v1/{projectId}/zones/{zoneId}", 200, true, nil, nil),
	}
	for index := range operations {
		operations[index].Mutation = operations[index].Method != http.MethodGet
		operations[index].Destructive = operations[index].Method == http.MethodDelete
	}
	return operations
}

func ancillary(parent, use, tag, operationID, method, path string, successStatus int, responseBody bool, queries []ancillaryQueryParameter, body *ancillaryBodyContract) ancillaryOperation {
	return ancillaryOperation{
		Parents:       []string{parent},
		Use:           use,
		Short:         method + " " + path,
		Tag:           tag,
		OperationID:   operationID,
		Method:        method,
		Path:          path,
		ProjectScope:  "projectId",
		Queries:       queries,
		Body:          body,
		SuccessStatus: successStatus,
		ResponseBody:  responseBody,
	}
}

func ancillaryWithProjectScope(op ancillaryOperation, parameter string) ancillaryOperation {
	op.ProjectScope = parameter
	return op
}

func ancillaryBody(name string, requiredFields ...string) *ancillaryBodyContract {
	return &ancillaryBodyContract{Name: name, RequiredFields: requiredFields}
}

func ancillaryWithPathKind(op ancillaryOperation, name string, kind ancillaryPathKind) ancillaryOperation {
	op.PathKinds = map[string]ancillaryPathKind{name: kind}
	return op
}
