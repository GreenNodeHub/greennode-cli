# delete-addon

## Description

Delete an add-on from a VKS cluster. Before executing, the command fetches and displays the add-on that will be deleted (name, version, status).

Unless `--force` is provided, you are prompted to confirm. Use `--dry-run` to see the preview without deleting anything.

With `--preserve`, VKS stops managing the add-on but keeps its Helm release and workloads in the cluster. Without it, the release is uninstalled.

The add-on goes to `DELETING`. Use [`wait addon-deleted`](wait.md#addon-deleted) to wait until it is gone.

## Synopsis

```
grn vks delete-addon
    --cluster-id <value>
    --addon-name <value>
    [--preserve]
    [--dry-run]
    [--force]
```

## Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--addon-name`** (string)

Name of the add-on to delete.

- Required: Yes

**`--preserve`** (boolean)

Keep the add-on's Helm release in the cluster.

- Required: No
- Default: `false`

**`--dry-run`** (boolean)

Display the add-on that would be deleted without sending the request.

- Required: No
- Default: `false`

**`--force`** (boolean)

Skip the confirmation prompt.

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Delete an add-on and wait until it is gone:

```bash
grn vks delete-addon --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi --force
grn vks wait addon-deleted --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi
```

Stop managing an add-on but keep it running:

```bash
grn vks delete-addon --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi --preserve
```
