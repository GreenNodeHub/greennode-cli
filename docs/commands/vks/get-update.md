# get-update

## Description

Get the status of an update: `InProgress`, `Successful`, `Failed` or `Cancelled`, with its parameters and errors.

The scope must match the one the update was recorded with: pass `--addon-name` for an update returned by [update-addon](update-addon.md), `--nodegroup-id` for a node group update, and neither for a cluster update.

## Synopsis

```
grn vks get-update
    --cluster-id <value>
    --update-id <value>
    [--nodegroup-id <value> | --addon-name <value>]
```

## Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--update-id`** (string)

ID of the update.

- Required: Yes

**`--addon-name`** (string)

Add-on the update belongs to.

- Required: No

**`--nodegroup-id`** (string)

Node group the update belongs to.

- Required: No

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

```bash
grn vks get-update \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --addon-name block-storage-csi \
  --update-id 0b3c7a8e-1f2d-4c5b-9a6e-7d8f9e0a1b2c
```
