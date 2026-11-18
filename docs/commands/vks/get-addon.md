# get-addon

## Description

Get the details of an add-on installed on a VKS cluster: version, status, health issues, configuration values and namespace.

Add-on status values:

| Status | Meaning |
|---|---|
| `CREATING` | The add-on is being installed |
| `ACTIVE` | Installed and its workloads are ready |
| `DEGRADED` | Installed, but its workloads are not ready yet (for example the cluster has no nodes). Turns `ACTIVE` on its own |
| `UPDATING` | An update is in progress |
| `DELETING` | Being deleted |
| `CREATE_FAILED`, `UPDATE_FAILED`, `DELETE_FAILED` | The operation failed; see `health.issues` |

## Synopsis

```
grn vks get-addon
    --cluster-id <value>
    --addon-name <value>
```

## Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--addon-name`** (string)

Name of the add-on.

- Required: Yes

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

```bash
grn vks get-addon --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi
```
