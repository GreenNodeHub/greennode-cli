# list-addons

## Description

List the add-ons installed on a VKS cluster. By default only the add-on names are returned (`addons`); with `--detail` the full add-ons are returned in `addonDetails` too.

## Synopsis

```
grn vks list-addons
    --cluster-id <value>
    [--detail]
```

## Options

**`--cluster-id`** (string)

ID of the cluster.

- Required: Yes

**`--detail`** (boolean)

Also return the full add-ons (status, version, health, configuration values, …).

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

List the add-on names of a cluster:

```bash
grn vks list-addons --cluster-id cls-abc12345-6789-def0-1234-abcdef012345
```

Show the status of each add-on:

```bash
grn vks list-addons --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 --detail \
  --query 'addonDetails[].[addonName,addonVersion,status]' --output table
```
