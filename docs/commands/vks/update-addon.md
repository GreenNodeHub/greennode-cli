# update-addon

## Description

Update the version or the configuration values of an add-on installed on a VKS cluster. At least one of `--addon-version` and `--configuration-values` is required.

Only the flags you set are sent: an omitted `--addon-version` or `--configuration-values` keeps the current one, and `--configuration-values ""` clears the custom values.

The update is asynchronous. The command returns an update with status `InProgress`; wait for it with [`wait update-successful`](wait.md#update-successful) or check it with [get-update](get-update.md) (pass `--addon-name` to both).

## Synopsis

```
grn vks update-addon
    --cluster-id <value>
    --addon-name <value>
    [--addon-version <value>]
    [--configuration-values <value>]
    [--resolve-conflicts <value>]
    [--client-request-token <value>]
    [--dry-run]
```

## Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--addon-name`** (string)

Name of the add-on.

- Required: Yes

**`--addon-version`** (string)

Target version, for example `1.3.0-vks.1`.

- Required: No
- Default: keep the current version

**`--configuration-values`** (string)

Configuration values as a JSON or YAML string, or `file://<path>`. At most 64 KiB. `""` clears the values.

- Required: No
- Default: keep the current values

**`--resolve-conflicts`** (string)

How to treat fields changed outside VKS: `NONE`, `OVERWRITE` or `PRESERVE` (case-insensitive).

- Required: No

**`--client-request-token`** (string)

Idempotency key, unique per cluster (1-64 characters).

- Required: No

**`--dry-run`** (boolean)

Print the request body without sending it.

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Upgrade an add-on and wait for the update:

```bash
UPDATE_ID=$(grn vks update-addon \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --addon-name block-storage-csi \
  --addon-version 1.3.0-vks.1 \
  --query update.id --output text)
grn vks wait update-successful \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --addon-name block-storage-csi \
  --update-id "$UPDATE_ID"
```

Clear the custom configuration values:

```bash
grn vks update-addon --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi --configuration-values ""
```
