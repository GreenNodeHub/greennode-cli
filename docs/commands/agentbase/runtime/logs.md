# runtime logs

Search a runtime's logs.

## Description

Search a runtime's logs at the runtime level (not per-endpoint)
(POST `/agent-runtimes/{id}/logs`). Returns `{totalCount, logs:[...]}`.

## Synopsis

```text
grn agentbase runtime logs <id>
    [--from <offset>] [--limit <value>]
    [--from-timestamp <RFC3339>] [--to-timestamp <RFC3339>]
    [--query <value>] [--order <value>]
```

## Arguments

**`<id>`** (string) — runtime id. Required.

## Options

**`--from`** (integer) — offset (max 5000). Default `0`.
**`--limit`** (integer) — max lines (max 500). Default `100`.
**`--from-timestamp`** (string) — `fromTimestamp` (RFC3339).
**`--to-timestamp`** (string) — `toTimestamp` (RFC3339).
**`--query`** (string) — log query filter.
**`--order`** (string) — sort order.

## Examples

```bash
grn agentbase runtime logs rt-1 --query "panic" --limit 20
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
