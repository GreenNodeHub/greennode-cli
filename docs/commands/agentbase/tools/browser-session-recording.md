# tools browser session recording

Download a browser session's recording.

## Description

Download a session's recording
(GET `/v1/browsers/{id}/sessions/{sessionId}/recording`,
`application/x-ndjson`).

The recording only exists **after the session is deleted** — the server
archives it on delete; until then the endpoint returns 404. So the flow is:
delete the session, then download the recording.

## Synopsis

```text
grn agentbase tools browser session recording <browser-id> <session-id>
    [--end-user-id <value>]
    [--output-file <path>]
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

**`<session-id>`** (string) — session id. Required.

## Options

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header), 1–128 chars of `A-Za-z0-9._:@-` (validated client-side).

**`--output-file`** (string) — write the recording to this file instead of
stdout.

## Output

- **Without `--output-file`** — the raw NDJSON bytes go to **stdout
  verbatim**, regardless of `-o` (this is a binary-ish download, not a
  rendered resource). Pipe it: `... recording br-1 bs-1 > session.ndjson`.
- **With `--output-file`** — the file is written (mode 0644), a summary line
  goes to stderr, and `-o id` prints the path; `-o json` prints
  `{"path": "...", "bytes": N}`.

## Examples

```bash
# After the session is deleted:
grn agentbase tools browser session recording br-1a2b3c bs-4d5e6f > session.ndjson
grn agentbase tools browser session recording br-1a2b3c bs-4d5e6f --output-file session.ndjson -o id
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
