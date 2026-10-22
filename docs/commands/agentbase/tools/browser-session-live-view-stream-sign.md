# tools browser session live-view-stream-sign

Sign a browser session's live-view stream URL.

## Description

Request a short-lived signed URL for a session's live-view stream (screenshots
over SSE) — POST `/v1/browsers/{id}/sessions/{sessionId}/live-view-stream-sign`.

The signed URL carries its own auth, so it can be opened directly (e.g. in a
browser tab or `curl`) without the IAM token. It expires after `expiresIn`
seconds.

## Synopsis

```text
grn agentbase tools browser session live-view-stream-sign <browser-id> <session-id>
    [--end-user-id <value>]
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

**`<session-id>`** (string) — session id. Required.

## Options

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header), 1–128 chars of `A-Za-z0-9._:@-` (validated client-side).

## Output

- `table` — Signed URL + Expires In (seconds)
- `json` — the raw `{url, expiresIn}` object
- `id` — the signed URL

## Examples

```bash
grn agentbase tools browser session live-view-stream-sign br-1a2b3c bs-4d5e6f
grn agentbase tools browser session live-view-stream-sign br-1a2b3c bs-4d5e6f -o id
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
