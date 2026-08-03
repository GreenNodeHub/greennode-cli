# runtime versions

List a runtime's versions.

## Description

List a runtime's versions (GET `/agent-runtimes/{id}/versions`), 1-based
paginated. Each row is the full spec of a version (image, command/args/env,
network, inbound auth, autoscaling).

## Synopsis

```text
grn agentbase runtime versions <id>
    [--page <value>]
    [--size <value>]
```

## Arguments

**`<id>`** (string) — runtime id. Required.

## Options

**`--page`** (integer) — page number (1-based). Default `1`.
**`--size`** (integer) — page size. Default `10`.

## Examples

```bash
grn agentbase runtime versions rt-1 -o json
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
