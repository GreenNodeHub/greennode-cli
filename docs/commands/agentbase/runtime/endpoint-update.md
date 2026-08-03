# runtime endpoint update

Roll a runtime endpoint to a version.

## Description

Roll an endpoint to a target version
(PUT `/agent-runtimes/{id}/endpoints/{endpointId}?version=<value>`). No body.
The service also exposes a deprecated PATCH variant; this PUT is canonical and
the PATCH QC row maps to it.

## Synopsis

```text
grn agentbase runtime endpoint update <id> <endpoint-id> --version <value>
```

## Arguments

**`<id>`** (string) — runtime id. Required.
**`<endpoint-id>`** (string) — endpoint id. Required.

## Options

**`--version`** (integer) — target version (required, positive).

## Examples

```bash
grn agentbase runtime endpoint update rt-1 ep-1 --version 3
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
