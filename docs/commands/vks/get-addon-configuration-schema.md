# get-addon-configuration-schema

## Description

Get the JSON Schema (draft-07) that `--configuration-values` of [create-addon](create-addon.md) and [update-addon](update-addon.md) must satisfy for one add-on version.

The schema is returned as a string in `configurationSchema`.

## Synopsis

```
grn vks get-addon-configuration-schema
    --addon-name <value>
    --addon-version <value>
```

## Options

**`--addon-name`** (string)

Name of the add-on.

- Required: Yes

**`--addon-version`** (string)

Version of the add-on, for example `1.2.0-vks.1`. Versions come from [list-addon-versions](list-addon-versions.md).

- Required: Yes

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Print the schema of an add-on version:

```bash
grn vks get-addon-configuration-schema \
  --addon-name block-storage-csi \
  --addon-version 1.2.0-vks.1 \
  --query configurationSchema --output text
```
