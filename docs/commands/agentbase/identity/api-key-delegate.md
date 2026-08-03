# identity api-key delegate

Authorize a delegated API key against a provider.

## Description

Authorize a delegated API key against a delegated API key provider
(POST `/api-key/delegate/{providerId}`, operation `authorizeApiKey`). The path
is on the identity root (no `/api/v1` prefix) — it is a user-agent redirect
surface like the OAuth callback paths. The provider returns a `redirectUrl`
(the caller's user-agent continues delegation there) plus a `success`/`message`
outcome. `--state` is the required query param; `--api-key` is the required
request body.

## Synopsis

```text
grn agentbase identity api-key delegate <provider-id>
    --state <value> --api-key <value>
```

## Arguments

**`<provider-id>`** (string) — delegated API key provider id. Required.

## Options

**`--state`** (string) — state query parameter (required).
**`--api-key`** (string) — API key to authorize (required).

## Examples

```bash
grn agentbase identity api-key delegate prov-1 --state abc-state --api-key ak-secret
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
