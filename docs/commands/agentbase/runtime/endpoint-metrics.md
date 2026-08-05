# runtime endpoint metrics

Fetch an endpoint's metrics.

## Description

Fetch CPU/memory metrics for an endpoint over a time range
(GET `/agent-runtimes/{id}/endpoints/{endpointId}/metrics` with
`?fromTimestamp=&toTimestamp=`). Returns CPU (double) and memory-bytes (long)
sample series.

## Synopsis

```text
grn agentbase runtime endpoint metrics <id> <endpoint-id>
    [--from <RFC3339>]
    [--to <RFC3339>]
```

## Arguments

**`<id>`** / **`<endpoint-id>`** — runtime / endpoint ids. Both required.

## Options

**`--from`** (string) — `fromTimestamp` (RFC3339).
**`--to`** (string) — `toTimestamp` (RFC3339).

## Examples

```bash
grn agentbase runtime endpoint metrics rt-1 ep-1 --from 2026-01-01T00:00:00Z -o json
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
