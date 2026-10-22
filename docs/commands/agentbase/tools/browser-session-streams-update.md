# tools browser session streams update

Toggle a browser session's automation stream.

## Description

Update a session's streams
(PUT `/v1/browsers/{id}/sessions/{sessionId}/streams`) — currently the
**automation stream** status: `ENABLED` or `DISABLED`.

The live-view stream is always on and cannot be toggled. The API returns no
body, so the CLI re-reads the session and renders the resulting streams.

## Synopsis

```text
grn agentbase tools browser session streams update <browser-id> <session-id>
    --status <ENABLED|DISABLED>
    [--end-user-id <value>]
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

**`<session-id>`** (string) — session id. Required.

## Options

**`--status`** (string) — the automation stream status. Required; one of
`ENABLED`, `DISABLED` (validated client-side).

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header), 1–128 chars of `A-Za-z0-9._:@-` (validated client-side).

## Output

- `table` — the re-read session fields + `Streams:` table
- `json` — the raw session detail object
- `id` — the session's id

## Examples

```bash
grn agentbase tools browser session streams update br-1a2b3c bs-4d5e6f --status ENABLED
grn agentbase tools browser session streams update br-1a2b3c bs-4d5e6f --status DISABLED
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
