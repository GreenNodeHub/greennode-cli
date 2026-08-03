# gateway access-logs stats

Aggregate access-log stats for a gateway.

## Description

Return aggregate access-log stats for a gateway
(GET `/api/v1/gateways/{name}/access-logs/stats`): total requests, success/error
rates, duration stats, a status histogram, a time series, and top tools/targets/
callers/user-agents.

## Synopsis

```text
grn agentbase gateway access-logs stats <name>
    [--from <value>] [--to <value>]
    [--mcp-method <value>] [--tool-name <value>]
    [--target-name <value>] [--http-status <value>] [--client-ip <value>]
    [--interval <value>] [--top-n <value>]
```

## Arguments

**`<name>`** (string) — gateway name. Required.

## Options

The filter options mirror `access-logs list` (`--from`/`--to`/`--mcp-method`/
`--tool-name`/`--target-name`/`--http-status`/`--client-ip`). Plus:

**`--interval`** (string) — time-series bucket interval (e.g. `1h`).
**`--top-n`** (integer) — number of top tools/targets/callers to return. Default `5`.

## Examples

```bash
grn agentbase gateway access-logs stats my-gw --interval 1h --top-n 10
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
