# memory event delete

Delete a session event.

## Description

Delete a single event from a session
(DELETE `/memories/{id}/actors/{actorId}/sessions/{sessionId}/events/{eventId}`).
200 OK; prints the deleted event id.

## Synopsis

```text
grn agentbase memory event delete <id> <actor-id> <session-id> <event-id>
```

## Arguments

**`<id>`** / **`<actor-id>`** / **`<session-id>`** / **`<event-id>`** —
memory / actor / session / event ids. All required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase memory event delete mem-1 act-1 ses-1 evt-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
