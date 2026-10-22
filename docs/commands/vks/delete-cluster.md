# delete-cluster

## Description

Delete a VKS cluster and all of its associated node groups. Before a live deletion, the command fetches and displays the cluster name, status, version, node count, and the list of node groups that will be removed.

Unless `--force` is provided, you are prompted to confirm. Use `--dry-run` to validate the cluster ID and display the DELETE request target without loading credentials, fetching resources, prompting, or deleting anything.

**This action is irreversible.**

## Synopsis

```
grn vks delete-cluster
    --cluster-id <value>
    [--dry-run]
    [--force]
```

## Options

**`--cluster-id`** (string)

ID of the cluster to delete.

- Required: Yes

**`--dry-run`** (boolean)

Validate the cluster ID and display the DELETE request target without sending the request.

- Required: No
- Default: `false`

**`--force`** (boolean)

Skip the interactive confirmation prompt and delete immediately. Useful in non-interactive scripts.

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Delete a cluster interactively (prompts for confirmation):

```bash
grn vks delete-cluster --cluster-id cls-abc12345-6789-def0-1234-abcdef012345
```

Preview the delete request without deleting:

```bash
grn vks delete-cluster \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --dry-run
```

Delete without confirmation (for use in scripts):

```bash
grn vks delete-cluster \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --force
```
