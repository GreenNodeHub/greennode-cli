# create-addon

## Description

Install an add-on on a VKS cluster. The request is accepted while the cluster is `WAITING_*`, `CREATING` or `ACTIVE`, so add-ons can be requested right after [create-cluster](create-cluster.md).

The add-on is returned with status `CREATING`. Use [`wait addon-active`](wait.md#addon-active) to wait until it is `ACTIVE`.

Find add-on names and versions with [list-addon-versions](list-addon-versions.md), and the accepted configuration values with [get-addon-configuration-schema](get-addon-configuration-schema.md).

## Synopsis

```
grn vks create-addon
    --cluster-id <value>
    --addon-name <value>
    [--addon-version <value>]
    [--configuration-values <value>]
    [--namespace <value>]
    [--resolve-conflicts <value>]
    [--client-request-token <value>]
    [--dry-run]
```

## Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--addon-name`** (string)

Name of the add-on, for example `block-storage-csi`: 2-63 lowercase letters, digits or hyphens.

- Required: Yes

**`--addon-version`** (string)

Version to install, for example `1.2.0-vks.1`.

- Required: No
- Default: the default version for the cluster's Kubernetes minor version

**`--configuration-values`** (string)

Configuration values as a JSON or YAML string, or `file://<path>` to read them from a file. At most 64 KiB. Must match the configuration schema of the version.

- Required: No (Yes when the version has `requiresConfiguration: true`)

**`--namespace`** (string)

Kubernetes namespace to install the add-on into.

- Required: No
- Default: the default namespace of the version

**`--resolve-conflicts`** (string)

How to treat in-cluster resources that already exist: `NONE`, `OVERWRITE` or `PRESERVE` (case-insensitive).

- Required: No

**`--client-request-token`** (string)

Idempotency key, unique per cluster (1-64 characters). Retrying with the same token does not install the add-on twice.

- Required: No

**`--dry-run`** (boolean)

Print the request body without sending it.

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Install the default version and wait until it is active:

```bash
grn vks create-addon --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi
grn vks wait addon-active --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi
```

Install a given version with configuration values from a file:

```bash
grn vks create-addon \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --addon-name block-storage-csi \
  --addon-version 1.2.0-vks.1 \
  --configuration-values file://values.yaml
```

Preview the request:

```bash
grn vks create-addon --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi --dry-run
```
