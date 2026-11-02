# gateway target sync

Sync a target's tools/list from its live MCP endpoint.

## Description

Fetch the live tools/list from the target's MCP endpoint and persist it (POST
`/api/v1/gateways/{name}/targets/{target}/sync`).

Non-interactive auth (NONE/2LO, or 3LO with cached consent) returns the tools
directly. Interactive auth (3LO without consent, INBOUND_FORWARD) returns
`status=authorization_required` plus an `authorizeUrl` — open it in a browser;
on completion the gateway finishes tools/list server-side and redirects to
`--return-to` (or a plain done page).

Use [target tools](target-tools.md) to read the persisted snapshot without
re-syncing.

## Synopsis

```text
grn agentbase gateway target sync <gateway> <target>
    [--return-to <value>]
```

## Arguments

**`<gateway>`** (string) — gateway name. Required.

**`<target>`** (string) — target name on that gateway. Required.

## Options

**`--return-to`** (string)

URL the gateway redirects the browser to after interactive sync (optional).

- Required: No
- Default: (empty)

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase gateway target sync my-gw github
grn agentbase gateway target sync my-gw slack --return-to https://app.example.com/sync-done
```

Interactive (3LO) sync — the first call returns an authorize URL on stderr
when consent is required:

```bash
grn agentbase gateway target sync my-gw github
# Status: authorization_required
# Authorization URL: https://github.com/login/oauth/authorize?...  (open in a browser)
# ...after completing consent, re-run:
grn agentbase gateway target sync my-gw github
```
