# tools browser session list

List a browser's sessions.

## Description

List the sessions attached to a browser
(GET `/v1/browsers/{id}/sessions`), 1-based paginated.

## Synopsis

```text
grn agentbase tools browser session list <browser-id>
    [--end-user-id <value>]
    [--page <value>]
    [--size <value>]
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

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
grn agentbase tools browser session list br-1a2b3c
grn agentbase tools browser session list br-1a2b3c --end-user-id alice@example.com
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
