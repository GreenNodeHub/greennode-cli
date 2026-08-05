# gateway access-logs list

List a gateway's access-log entries.

## Description

List a gateway's access-log entries
(GET `/api/v1/gateways/{name}/access-logs`). The table view shows time, target,
MCP method/tool, upstream status, duration, and any error.

## Synopsis

```text
grn agentbase gateway access-logs list <name>
    [--from <value>] [--to <value>]
    [--mcp-method <value>] [--tool-name <value>]
    [--target-name <value>] [--http-status <value>] [--client-ip <value>]
    [--page <value>] [--page-size <value>]
```

## Arguments

**`<name>`** (string) — gateway name. Required.

## Options

**`--from`** (string) — filter: ISO8601 from (inclusive).
**`--to`** (string) — filter: ISO8601 to (exclusive).
**`--mcp-method`** (string) — filter: MCP method (e.g. tools/call).
**`--tool-name`** (string) — filter: MCP tool name.
**`--target-name`** (string) — filter: upstream target name.
**`--http-status`** (string) — filter: upstream HTTP status code.
**`--client-ip`** (string) — filter: caller client IP.
**`--page`** (integer) — page number (1-based). Default `1`.
**`--page-size`** (integer) — page size. Default `50`.

## Examples

```bash
grn agentbase gateway access-logs list my-gw --http-status 500
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
