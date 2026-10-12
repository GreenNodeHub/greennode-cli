# tools browser session get

Show a browser session.

## Description

Show a browser session with its streams
(GET `/v1/browsers/{id}/sessions/{sessionId}`).

The detail includes the stream endpoints: the CDP **automation stream**
(WebSocket), its status, and the **live-view stream** (screenshots over SSE).

## Synopsis

```text
grn agentbase tools browser session get <browser-id> <session-id>
    [--end-user-id <value>]
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

**`<session-id>`** (string) — session id. Required.

## Options

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header).

## Output

- `table` — session fields, then a `Streams:` table (Automation Stream,
  Automation Status, Live View Stream; `-` when absent)
- `json` — the raw session detail object
- `id` — the session's id

## Examples

```bash
grn agentbase tools browser session get br-1a2b3c bs-4d5e6f
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
