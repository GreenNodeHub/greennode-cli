# wait

## Description

Poll until a VKS cluster, node group, add-on or update reaches a desired state. Exits with code `255` if the resource reaches a terminal failure state (`ERROR`, `FAILED`, `*_FAILED`, or an update `Failed`/`Cancelled`) or the wait times out. Use after asynchronous operations such as create or delete.

`wait` is a parent command. Invoke one of its subcommands depending on the resource type and desired state:

| Subcommand | Waits until |
|---|---|
| `cluster-active` | Cluster status is `ACTIVE` |
| `cluster-deleted` | Cluster no longer exists (HTTP 404) |
| `nodegroup-active` | Node group status is `ACTIVE` |
| `nodegroup-deleted` | Node group no longer exists (HTTP 404) |
| `addon-active` | Add-on status is `ACTIVE` |
| `addon-deleted` | Add-on no longer exists (HTTP 404) |
| `update-successful` | Update status is `Successful` |

## cluster-active

### Synopsis

```
grn vks wait cluster-active
    --cluster-id <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

**`--cluster-id`** (string)

ID of the cluster to poll.

- Required: Yes

**`--delay`** (integer)

Seconds to wait between poll attempts.

- Required: No
- Default: `30`

**`--max-attempts`** (integer)

Maximum number of poll attempts before the waiter times out.

- Required: No
- Default: `40`

## cluster-deleted

### Synopsis

```
grn vks wait cluster-deleted
    --cluster-id <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

**`--cluster-id`** (string)

ID of the cluster to poll.

- Required: Yes

**`--delay`** (integer)

Seconds to wait between poll attempts.

- Required: No
- Default: `30`

**`--max-attempts`** (integer)

Maximum number of poll attempts before the waiter times out.

- Required: No
- Default: `40`

## nodegroup-active

### Synopsis

```
grn vks wait nodegroup-active
    --cluster-id <value>
    --nodegroup-id <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

**`--cluster-id`** (string)

ID of the cluster that owns the node group.

- Required: Yes

**`--nodegroup-id`** (string)

ID of the node group to poll.

- Required: Yes

**`--delay`** (integer)

Seconds to wait between poll attempts.

- Required: No
- Default: `30`

**`--max-attempts`** (integer)

Maximum number of poll attempts before the waiter times out.

- Required: No
- Default: `80`

## nodegroup-deleted

### Synopsis

```
grn vks wait nodegroup-deleted
    --cluster-id <value>
    --nodegroup-id <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

**`--cluster-id`** (string)

ID of the cluster that owns the node group.

- Required: Yes

**`--nodegroup-id`** (string)

ID of the node group to poll.

- Required: Yes

**`--delay`** (integer)

Seconds to wait between poll attempts.

- Required: No
- Default: `30`

**`--max-attempts`** (integer)

Maximum number of poll attempts before the waiter times out.

- Required: No
- Default: `40`

## addon-active

`CREATE_FAILED` and `UPDATE_FAILED` stop the wait with exit code `255`. `DEGRADED` (installed, workloads not ready yet — for example the cluster has no nodes) keeps the waiter polling, since the add-on turns `ACTIVE` on its own once its workloads run.

### Synopsis

```
grn vks wait addon-active
    --cluster-id <value>
    --addon-name <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--addon-name`** (string)

Name of the add-on to poll.

- Required: Yes

**`--delay`** (integer)

Seconds to wait between poll attempts.

- Required: No
- Default: `10`

**`--max-attempts`** (integer)

Maximum number of poll attempts before the waiter times out.

- Required: No
- Default: `60`

## addon-deleted

`DELETE_FAILED` stops the wait with exit code `255`.

### Synopsis

```
grn vks wait addon-deleted
    --cluster-id <value>
    --addon-name <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--addon-name`** (string)

Name of the add-on to poll.

- Required: Yes

**`--delay`** (integer)

Seconds to wait between poll attempts.

- Required: No
- Default: `10`

**`--max-attempts`** (integer)

Maximum number of poll attempts before the waiter times out.

- Required: No
- Default: `60`

## update-successful

`Failed` and `Cancelled` stop the wait with exit code `255`; see the errors with [get-update](get-update.md).

### Synopsis

```
grn vks wait update-successful
    --cluster-id <value>
    --update-id <value>
    [--addon-name <value> | --nodegroup-id <value>]
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--update-id`** (string)

ID of the update, as returned by `update-addon` (`update.id`).

- Required: Yes

**`--addon-name`** (string)

Add-on the update belongs to. Required for an add-on update.

- Required: No

**`--nodegroup-id`** (string)

Node group the update belongs to.

- Required: No

**`--delay`** (integer)

Seconds to wait between poll attempts.

- Required: No
- Default: `10`

**`--max-attempts`** (integer)

Maximum number of poll attempts before the waiter times out.

- Required: No
- Default: `60`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Wait for a cluster to become active after creation:

```bash
grn vks wait cluster-active \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345
```

Wait for a cluster to be fully deleted:

```bash
grn vks delete-cluster --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --force
grn vks wait cluster-deleted \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345
```

Wait for a node group to become active, polling every 15 seconds:

```bash
grn vks wait nodegroup-active \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --nodegroup-id ng-abc12345-6789-def0-1234-abcdef012345 \
  --delay 15
```

Wait for a node group to be deleted:

```bash
grn vks wait nodegroup-deleted \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --nodegroup-id ng-abc12345-6789-def0-1234-abcdef012345
```

Wait for an add-on to become active after `create-addon`:

```bash
grn vks wait addon-active \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --addon-name block-storage-csi
```

Wait for an add-on update to finish:

```bash
grn vks wait update-successful \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --addon-name block-storage-csi \
  --update-id 0b3c7a8e-1f2d-4c5b-9a6e-7d8f9e0a1b2c
```
