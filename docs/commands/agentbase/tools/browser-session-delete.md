# tools browser session delete

Delete a browser session.

## Description

Delete a browser session
(DELETE `/v1/browsers/{id}/sessions/{sessionId}`).

Deleting a session is what makes its **recording** available: the recording
is archived server-side on delete, so
[`browser session recording`](browser-session-recording.md) can only download
it **after** the session is deleted (it returns 404 until then).

## Synopsis

```text
grn agentbase tools browser session delete <browser-id> <session-id>
    [--end-user-id <value>]
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

**`<session-id>`** (string) — session id. Required.

## Options

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header), 1–128 chars of `A-Za-z0-9._:@-` (validated client-side).

## Output

- `table` — nothing on stdout (a confirmation line goes to stderr)
- `json` — `{"id": "<session-id>"}`
- `id` — the deleted session's id

## Examples

```bash
grn agentbase tools browser session delete br-1a2b3c bs-4d5e6f
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
