# tools code-interpreter session delete

Delete a code interpreter session.

## Description

Delete a code interpreter session
(DELETE `/v1/code-interpreters/{id}/sessions/{sessionId}`).

## Synopsis

```text
grn agentbase tools code-interpreter session delete <code-interpreter-id> <session-id>
    [--end-user-id <value>]
```

## Arguments

**`<code-interpreter-id>`** (string) — code interpreter id. Required.

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
grn agentbase tools code-interpreter session delete ci-1a2b3c cs-4d5e6f
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
