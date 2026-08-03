# memory event create

Create a session event.

## Description

Append an event to a session
(POST `/memories/{id}/actors/{actorId}/sessions/{sessionId}/events`). The event
payload carries type/role/message/binary-data; `--event-timestamp` sets the
event time.

## Synopsis

```text
grn agentbase memory event create <id> <actor-id> <session-id>
    --type <value>
    [--role <value>]
    [--message <value>]
    [--binary-data <value>]
    [--event-timestamp <value>]
```

## Arguments

**`<id>`** / **`<actor-id>`** / **`<session-id>`** — memory / actor / session
ids. All required.

## Options

**`--type`** (string) — event type (required).
**`--role`** (string) — event role.
**`--message`** (string) — event message (max 100k chars).
**`--binary-data`** (string) — event binary data (max ~10 MiB).
**`--event-timestamp`** (string) — event timestamp (RFC3339).

## Examples

```bash
grn agentbase memory event create mem-1 act-1 ses-1 --type MESSAGE --role user --message "hello"
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
