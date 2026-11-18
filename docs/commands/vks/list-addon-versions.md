# list-addon-versions

## Description

List the add-ons of the VKS catalog and their versions. Use it to find the add-on names and versions accepted by [create-addon](create-addon.md) and [update-addon](update-addon.md).

Without `--k8s-version` the whole catalog is returned. Versions are listed newest first. For each Kubernetes minor version, the version with `defaultVersion: true` in `compatibilities` is the one `create-addon` installs when `--addon-version` is omitted. A version with `requiresConfiguration: true` cannot be installed without `--configuration-values`.

`DEPRECATED` versions are returned only with `--include-deprecated`.

## Synopsis

```
grn vks list-addon-versions
    [--addon-name <value>]
    [--k8s-version <value>]
    [--owners <value>]
    [--publishers <value>]
    [--types <value>]
    [--include-deprecated]
```

## Options

**`--addon-name`** (string)

Return only this add-on.

- Required: No

**`--k8s-version`** (string)

Return only versions compatible with this Kubernetes minor (`1.32`) or version (`1.32.9`). A leading `v` is accepted.

- Required: No

**`--owners`** (string)

Comma-separated owners: `vks`, `community`, `marketplace`.

- Required: No

**`--publishers`** (string)

Comma-separated publishers.

- Required: No

**`--types`** (string)

Comma-separated types: `networking`, `dns`, `storage`, `loadbalancing`, `observability`, `security`, `gitops`, `other`.

- Required: No

**`--include-deprecated`** (boolean)

Also return `DEPRECATED` versions.

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

List the add-ons available for Kubernetes 1.32:

```bash
grn vks list-addon-versions --k8s-version 1.32
```

List the versions of one add-on:

```bash
grn vks list-addon-versions --addon-name block-storage-csi \
  --query 'addons[0].addonVersions[].addonVersion' --output text
```
