# tools code-interpreter session create

Create a code interpreter session.

## Description

Create a session on a code interpreter
(POST `/v1/code-interpreters/{id}/sessions`).

The code interpreter must be `ACTIVE` — run
[`code-interpreter wait`](code-interpreter-wait.md) first; otherwise the API
rejects the session with 409. Session creation itself is **synchronous**: the
response is the ready session.

## Synopsis

```text
grn agentbase tools code-interpreter session create <code-interpreter-id>
    [--name <value>]
    [--end-user-id <value>]
```

## Arguments

**`<code-interpreter-id>`** (string) — code interpreter id. Required.

## Options

**`--name`** (string) — session name. Optional (the server defaults it).

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header, 1–128 chars of `A-Za-z0-9._:@-`, validated client-side).

## Output

- `table` — the new session (field/value)
- `json` — the raw session object
- `id` — the new session's id

On success a hint to run `session invoke` is printed to stderr.

## Examples

```bash
grn agentbase tools code-interpreter session create ci-1a2b3c
grn agentbase tools code-interpreter session create ci-1a2b3c --name analysis --end-user-id alice@example.com
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
