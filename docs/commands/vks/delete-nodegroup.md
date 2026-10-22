# delete-nodegroup

## Description

Delete a specific node group from a VKS cluster. Before a live deletion, the command fetches and displays the name, status, and node count of the node group that will be removed.

Unless `--force` is provided, you are prompted to confirm. Use `--dry-run` to validate the identifiers and display the DELETE request target without loading credentials, fetching the node group, prompting, or deleting anything. Use `--force-delete` to include `forceDelete=true` in that preview and instruct the API to perform a forced deletion during live execution.

**This action is irreversible.**

## Synopsis

```
grn vks delete-nodegroup
    --cluster-id <value>
    --nodegroup-id <value>
    [--force-delete]
    [--dry-run]
    [--force]
```

## Options

**`--cluster-id`** (string)

ID of the cluster that owns the node group.

- Required: Yes

**`--nodegroup-id`** (string)

ID of the node group to delete.

- Required: Yes

**`--force-delete`** (boolean)

Instruct the API to perform a forced deletion of the node group. Passes `forceDelete=true` as a query parameter to the delete endpoint.

- Required: No
- Default: `false`

**`--dry-run`** (boolean)

Validate the identifiers and display the DELETE request target without sending the request.

- Required: No
- Default: `false`

**`--force`** (boolean)

Skip the interactive confirmation prompt and delete immediately. Useful in non-interactive scripts.

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Delete a node group interactively (prompts for confirmation):

```bash
grn vks delete-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --nodegroup-id ng-abc12345-6789-def0-1234-abcdef012345
```

Preview the delete request without deleting:

```bash
grn vks delete-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --nodegroup-id ng-abc12345-6789-def0-1234-abcdef012345 \
  --dry-run
```

Delete without confirmation (for use in scripts):

```bash
grn vks delete-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --nodegroup-id ng-abc12345-6789-def0-1234-abcdef012345 \
  --force
```

Force-delete a stuck node group without confirmation:

```bash
grn vks delete-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --nodegroup-id ng-abc12345-6789-def0-1234-abcdef012345 \
  --force-delete \
  --force
```
