# tools browser session create

Create a browser session.

## Description

Create a session on a browser (POST `/v1/browsers/{id}/sessions`).

The browser must be `ACTIVE` — run [`browser wait`](browser-wait.md) first;
otherwise the API rejects the session with 409. Session creation itself is
**synchronous**: the response is the ready session.

## Synopsis

```text
grn agentbase tools browser session create <browser-id>
    [--name <value>]
    [--end-user-id <value>]
```

## Arguments

**`<browser-id>`** (string) — browser id. Required.

## Options

**`--name`** (string) — session name. Optional (the server defaults it).

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header, 1–128 chars of `A-Za-z0-9._:@-`, validated server-side).

## Output

- `table` — the new session (field/value)
- `json` — the raw session object
- `id` — the new session's id

On success a hint to fetch the session detail (for the stream endpoints) is
printed to stderr.

## Examples

```bash
grn agentbase tools browser session create br-1a2b3c
grn agentbase tools browser session create br-1a2b3c --name checkout-flow --end-user-id alice@example.com
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
