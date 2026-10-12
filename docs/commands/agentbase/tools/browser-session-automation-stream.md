# tools browser session automation-stream

Print a browser session's automation stream URL.

## Description

Print the CDP automation stream (WebSocket) URL of a browser session:
`{endpoint}/v1/browsers/{id}/sessions/{sessionId}/automation-stream`.

This is a **client-side print only** — no HTTP request is made, so it works
offline and prints the URL for the **active environment** (dev/prod). The
actual stream endpoint is the same URL fetched (GET, unauthenticated relay).

The stream must be enabled for the session — see
[`browser session streams update`](browser-session-streams-update.md) to
toggle it.

## Synopsis

```text
grn agentbase tools browser session automation-stream <browser-id> <session-id>
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

**`<session-id>`** (string) — session id. Required.

## Output

- `table` — one row (Automation Stream URL)
- `json` — `{"url": "..."}`
- `id` — the bare URL (script-friendly)

## Examples

```bash
grn agentbase tools browser session automation-stream br-1a2b3c bs-4d5e6f -o id
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
