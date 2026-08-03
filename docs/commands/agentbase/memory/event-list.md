# memory event list

List a session's events.

## Description

List the events recorded for a session
(GET `/memories/{id}/actors/{actorId}/sessions/{sessionId}/events`). The
endpoint streams events with no fixed response schema, so the page size is
capped at 100. `--from`/`--to` are RFC3339 timestamps.

## Synopsis

```text
grn agentbase memory event list <id> <actor-id> <session-id>
    [--from <value>]
    [--to <value>]
    [--page <value>]
    [--size <value>]
```

## Arguments

**`<id>`** (string) — memory id. Required.
**`<actor-id>`** (string) — actor id. Required.
**`<session-id>`** (string) — session id. Required.

## Options

**`--from`** (string) — `fromTimestamp` (RFC3339).
**`--to`** (string) — `toTimestamp` (RFC3339).
**`--page`** (integer) — page number (1-based). Default `1`.
**`--size`** (integer) — page size (capped at 100). Default `100`.

## Examples

```bash
grn agentbase memory event list mem-1 act-1 ses-1 --from 2026-01-01T00:00:00Z
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
