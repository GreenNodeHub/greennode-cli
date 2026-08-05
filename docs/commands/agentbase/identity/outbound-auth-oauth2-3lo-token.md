# identity outbound-auth oauth2 3lo-token

Get a 3-legged OAuth2 token.

## Description

Retrieve a 3-legged OAuth2 token for an agent identity via an OAuth2 provider.

Required flags: `--agent-user-id`, `--return-url`, `--scope`.
Optional flags: `--session-id`, `--custom-parameters`, `--custom-state`, `--force-authentication`.

The response may include the access token and an authorization URL. Use `-o json`
(or `-o id`) to reveal the full values; the default `table` output is for human inspection.

## Synopsis

```text
grn agentbase identity outbound-auth oauth2 3lo-token <provider-name> <identity-name>
    --agent-user-id <value>
    --return-url <value>
    --scope <value>
    [--session-id <value>]
    [--custom-parameters <value>]
    [--custom-state <value>]
    [--force-authentication]
```

## Arguments

- `<provider-name>` — name of the OAuth2 provider
- `<identity-name>` — name of the agent identity

## Options

**`--agent-user-id`** (string)

Agent user ID (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)

**`--return-url`** (string)

Return URL after authorization (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)

**`--scope`** (list&lt;string&gt;)

OAuth2 scope (repeatable, required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted as a comma-separated list when `--interactive` is set)
- Syntax: repeat the flag, e.g. `--scope openid --scope email`

**`--session-id`** (string)

Session ID (UUID format).

- Required: No
- Default: _(empty)_

**`--custom-parameters`** (string)

Custom parameters as a JSON object, e.g. `'{"key1":"value1"}'`.

- Required: No
- Default: _(empty)_
- Constraints: must be valid JSON object with string values; an invalid JSON value is rejected.

**`--custom-state`** (string)

Custom state parameter.

- Required: No
- Default: _(empty)_

**`--force-authentication`** (boolean)

Force re-authentication.

- Required: No
- Default: `false`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Get a 3-legged token with scopes (table view):

```bash
grn agentbase identity outbound-auth oauth2 3lo-token \
  my-oauth2-provider my-agent \
  --agent-user-id user-123 \
  --return-url https://app.example.com/callback \
  --scope openid --scope email
```

Pass custom parameters as JSON and reveal the full response (prints secret values):

```bash
grn agentbase identity outbound-auth oauth2 3lo-token \
  my-oauth2-provider my-agent \
  --agent-user-id user-123 \
  --return-url https://app.example.com/callback \
  --scope openid \
  --custom-parameters '{"prompt":"consent"}' \
  -o json
```
