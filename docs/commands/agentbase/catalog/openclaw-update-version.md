# catalog openclaw update-version

Roll an OpenClaw workspace to a version.

## Description

Roll an OpenClaw workspace to a target version (PUT `/v1/openclaws/{id}/version`
with `?versionId=<value>`). The service also exposes a deprecated PATCH
variant; this PUT is canonical and the PATCH QC row maps to it.

## Synopsis

```text
grn agentbase catalog openclaw update-version <id> --version-id <value>
```

## Arguments

**`<id>`** (string)

Id of the OpenClaw workspace to roll.

- Required: Yes (exactly one positional argument)

## Options

**`--version-id`** (string)

Target version id (required).

- Required: Yes

## Examples

```bash
grn agentbase catalog openclaw update-version oc-123 --version-id v2
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
