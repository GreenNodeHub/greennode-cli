# runtime endpoint create

Create a runtime endpoint.

## Description

Create a new endpoint on a runtime (POST `/agent-runtimes/{id}/endpoints`).
`--name` is required; `--version` is optional (defaults server-side, minimum 1).

## Synopsis

```text
grn agentbase runtime endpoint create <id> --name <value> [--version <value>]
```

## Arguments

**`<id>`** (string) — runtime id. Required.

## Options

**`--name`** (string) — endpoint name (required).
**`--version`** (integer) — target version (optional, defaults server-side).

## Examples

```bash
grn agentbase runtime endpoint create rt-1 --name ep-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
