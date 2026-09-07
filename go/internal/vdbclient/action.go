package vdbclient

// Resource types accepted by the action and resize endpoints. There is one per
// KIND of resource, and sending the wrong one is a validation error that does not
// name the field — so never hard-code "dbaas" for something that is not an
// instance.
const (
	ResourceTypeInstance      = "dbaas"
	ResourceTypeBackup        = "dbaas-backup"
	ResourceTypeBackupStorage = "dbaas-backup-storage"
)

// ActionBody builds the request for the action endpoints: instance start / stop /
// reboot / detach-replica / delete, and backup-storage delete.
//
// Four things about this shape are easy to get wrong:
//
//   - The resource ID goes in the BODY as well as the path, inside a list, under
//     the key "instancesId" (not "instanceId") — even when the resource is a backup
//     storage rather than an instance.
//   - The spec models all of these with one shared request class, so each of them
//     documents every action the family knows (start|stop|reboot|detach_replica).
//     That is an artifact: send the action that matches the path and nothing else.
//     Confirmed with the product team, 2026-08-13.
//   - The field is "resType" here but "resourceType" on the resize endpoints —
//     see ResizeBody.
//   - resourceType is NOT always "dbaas": pass the constant for the resource being
//     acted on.
//
// config carries the per-resource options the delete endpoints accept; pass nil
// when there are none.
func ActionBody(resourceType, resourceID, action string, config map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"databaseInstances": []interface{}{instanceDetail(resourceID, config)},
		"action":            action,
		"resType":           resourceType,
	}
}

// ResizeBody builds the request for the resize endpoints (instance resize-instance
// / resize-storage, backup-storage resize) and for backup restore, whose action is
// "restore_backup".
//
// Same shape as ActionBody except the resource-type field is spelled
// **resourceType**, not resType. Sending the wrong one is a 400 that reads like a
// validation error on something else.
func ResizeBody(resourceType, resourceID, action string, config map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"databaseInstances": []interface{}{instanceDetail(resourceID, config)},
		"action":            action,
		"resourceType":      resourceType,
	}
}

// ResizeConfigBody is ResizeBody for the one endpoint whose detail carries ONLY a
// config: backup restore. RestoreBackupDetail has no instancesId at all — the
// backup being restored is named inside the config, and the config otherwise
// describes the new instance to build.
func ResizeConfigBody(resourceType, action string, config map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"databaseInstances": []interface{}{map[string]interface{}{"config": config}},
		"action":            action,
		"resourceType":      resourceType,
	}
}

func instanceDetail(resourceID string, config map[string]interface{}) map[string]interface{} {
	detail := map[string]interface{}{"instancesId": resourceID}
	if config != nil {
		detail["config"] = config
	}
	return detail
}
