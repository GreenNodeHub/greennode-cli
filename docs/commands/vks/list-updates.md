# list-updates

## Description

List the IDs of the updates of a VKS cluster, newest first. Without a filter the updates of the cluster itself are returned; with `--nodegroup-id` those of that node group; with `--addon-name` those of that add-on. At most one filter can be set.

All pages are fetched unless `--max-results` or `--next-token` is set; then one page is returned with its `nextToken`.

## Synopsis

```
grn vks list-updates
    --cluster-id <value>
    [--nodegroup-id <value> | --addon-name <value>]
    [--max-results <value>]
    [--next-token <value>]
```

## Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--nodegroup-id`** (string)

Return only the updates of this node group.

- Required: No

**`--addon-name`** (string)

Return only the updates of this add-on.

- Required: No

**`--max-results`** (integer)

Page size, 1-100. Disables auto-pagination.

- Required: No

**`--next-token`** (string)

`nextToken` of the previous page. Disables auto-pagination.

- Required: No

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

List the updates of an add-on:

```bash
grn vks list-updates --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --addon-name block-storage-csi
```
