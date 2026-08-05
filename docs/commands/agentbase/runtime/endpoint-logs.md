# runtime endpoint logs

Search an endpoint's logs.

## Description

Search an endpoint's logs
(POST `/agent-runtimes/{id}/endpoints/{endpointId}/logs`) over a time range /
offset. Returns `{totalCount, logs:[{timestamp, content}]}`.

## Synopsis

```text
grn agentbase runtime endpoint logs <id> <endpoint-id>
    [--from <offset>] [--limit <value>]
    [--from-timestamp <RFC3339>] [--to-timestamp <RFC3339>]
    [--query <value>] [--order <value>]
```

## Arguments

**`<id>`** / **`<endpoint-id>`** — runtime / endpoint ids. Both required.

## Options

**`--from`** (integer) — offset (max 5000). Default `0`.
**`--limit`** (integer) — max lines (max 500). Default `100`.
**`--from-timestamp`** (string) — `fromTimestamp` (RFC3339).
**`--to-timestamp`** (string) — `toTimestamp` (RFC3339).
**`--query`** (string) — log query filter.
**`--order`** (string) — sort order.

## Examples

```bash
grn agentbase runtime endpoint logs rt-1 ep-1 --query "error" --limit 50
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
