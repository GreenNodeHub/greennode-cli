# runtime endpoint list

List a runtime's endpoints.

## Description

List the endpoints of a runtime (GET `/agent-runtimes/{id}/endpoints`),
1-based paginated. Each row tracks the endpoint version, target/live version
(a rolling update), replica count, URL, and status.

## Synopsis

```text
grn agentbase runtime endpoint list <id>
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
grn agentbase runtime endpoint list rt-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
