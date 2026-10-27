# outbound-auth managed 3lo-token

Get a 3-legged OAuth2 token via a managed provider.

## Description

Retrieve a 3-legged OAuth2 token for an agent identity via a
platform-managed OAuth2 provider (POST
`/api/v1/outbound-auth/managed-oauth2-providers/{providerName}/agent-identities/{agentIdentityName}/tokens/3lo`).
The flow is delegated to the browser: the first call returns an
`authorizationUrl` to open; the provider redirects to the managed provider's
fixed callback, and the token is completed server-side.

Unlike [outbound-auth oauth2 3lo-token](outbound-auth-oauth2-3lo-token.md),
no client credentials are involved — the managed provider holds them.

## Synopsis

```text
grn agentbase access outbound-auth managed 3lo-token <provider-name> <identity-name>
    --agent-user-id <value>
    --return-url <value>
    --scope <value>...
    [--session-id <value>]
    [--custom-parameters <json>]
    [--custom-state <value>]
    [--force-authentication]
```

## Arguments

**`<provider-name>`** (string) — managed provider name. Required.

**`<identity-name>`** (string) — agent identity name. Required.

## Options

**`--agent-user-id`** (string)

Agent user ID. Required (or use `--interactive`).

- Required: Yes
- Default: (empty)

**`--return-url`** (string)

Return URL after authorization. Required (or use `--interactive`).

- Required: Yes
- Default: (empty)

**`--scope`** (stringArray)

OAuth2 scope (repeatable). Required (or use `--interactive`).

- Required: Yes
- Default: (empty)
- Syntax: `--scope repo --scope read:org`

**`--session-id`** (string)

Session ID (UUID format).

- Required: No
- Default: (empty)

**`--custom-parameters`** (string)

Custom parameters as a JSON object, e.g. `'{"key1":"value1"}'`.

- Required: No
- Default: (empty)

**`--custom-state`** (string)

Custom state parameter.

- Required: No
- Default: (empty)

**`--force-authentication`** (bool)

Force re-authentication.

- Required: No
- Default: `false`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase access outbound-auth managed 3lo-token github my-agent \
  --agent-user-id user-123 \
  --return-url https://app.example.com/done \
  --scope repo
```

The first call typically returns `authorization_required` with an
authorization URL — open it in a browser; the token is completed server-side
after consent at the provider's fixed callback.
