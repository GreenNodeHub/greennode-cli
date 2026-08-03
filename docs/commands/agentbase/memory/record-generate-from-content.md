# memory record generate-from-content

Generate memory records from chat content.

## Description

Generate long-term memory records from arbitrary chat content
(POST `/memories/{id}/memory-records:generate-from-content` with
`?longTermMemoryStrategyId=&actorId=&sessionId=` and a body of
`{chatMessages: [{role, content}, ...]}`). Supply messages via `--message`
(repeatable; role defaults to user) or a `--file` with a chatMessages array
(`--file` is authoritative when set).

## Synopsis

```text
grn agentbase memory record generate-from-content <id>
    --strategy-id <value> [--actor-id <value>] [--session-id <value>]
    [--file <path> | --message <value> ...]
```

## Arguments

**`<id>`** (string) — memory id. Required.

## Options

**`--strategy-id`** (string) — long-term-memory strategy id (required).
**`--actor-id`** (string) — actor id (optional).
**`--session-id`** (string) — session id (optional).
**`--file`** (string) — spec file with a chatMessages array (authoritative when set).
**`--message`** (string, repeatable) — chat message content, role=user.

## Examples

```bash
grn agentbase memory record generate-from-content mem-1 --strategy-id strat-1 --message "preferred language: Go"
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
