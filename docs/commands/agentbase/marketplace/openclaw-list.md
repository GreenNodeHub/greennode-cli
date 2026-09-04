# marketplace openclaw list

List OpenClaw workspaces.

## Description

List OpenClaw workspaces (GET `/v1/openclaws`), 1-based paginated. The table
view shows the workspace id, name, version, status, and timestamps.

## Synopsis

```text
grn agentbase marketplace openclaw list
    [--page <value>]
    [--size <value>]
```

## Options

**`--page`** (integer)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (integer)

Page size.

- Required: No
- Default: `10`

## Examples

```bash
grn agentbase marketplace openclaw list -o json
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
