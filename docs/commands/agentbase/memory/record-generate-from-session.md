# memory record generate-from-session

Generate memory records from a session.

## Description

Generate long-term memory records from a session's events
(POST `/memories/{id}/memory-records:generate-from-session` with
`?actorId=&sessionId=&longTermMemoryStrategyId=`). No body; all three query
params are required. 200 OK on success.

## Synopsis

```text
grn agentbase memory record generate-from-session <id>
    --actor-id <value> --session-id <value> --strategy-id <value>
```

## Arguments

**`<id>`** (string) — memory id. Required.

## Options

**`--actor-id`** (string) — actor id (required).
**`--session-id`** (string) — session id (required).
**`--strategy-id`** (string) — long-term-memory strategy id (required).

## Examples

```bash
grn agentbase memory record generate-from-session mem-1 --actor-id act-1 --session-id ses-1 --strategy-id strat-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
