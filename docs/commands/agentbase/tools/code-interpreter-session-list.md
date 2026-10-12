# tools code-interpreter session list

List a code interpreter's sessions.

## Description

List the sessions attached to a code interpreter
(GET `/v1/code-interpreters/{id}/sessions`), 1-based paginated.

## Synopsis

```text
grn agentbase tools code-interpreter session list <code-interpreter-id>
    [--end-user-id <value>]
    [--page <value>]
    [--size <value>]
```

## Arguments

**`<code-interpreter-id>`** (string) — code interpreter id. Required.

## Options

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header, 1–128 chars of `A-Za-z0-9._:@-`, validated server-side).

**`--page`** (integer) — page number (1-based). Default `1`.

**`--size`** (integer) — page size. Default `10`.

## Output

- `table` — ID, Name, Status, End User, Created + a paging footer
- `json` — the raw `{listData, ...}` envelope
- `id` — the first session's id

## Examples

```bash
grn agentbase tools code-interpreter session list ci-1a2b3c
grn agentbase tools code-interpreter session list ci-1a2b3c --end-user-id alice@example.com
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
