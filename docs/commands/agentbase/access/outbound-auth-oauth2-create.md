# access outbound-auth oauth2 create

Create an OAuth2 provider.

## Description

Create a new OAuth2 provider. The name must be 3-50 characters and match the
pattern `^[a-zA-Z0-9_-]+$`. All of name, client-id, client-secret,
authorization-url, and token-url are required.

## Synopsis

```text
grn agentbase access outbound-auth oauth2 create
    --name <value>
    --client-id <value>
    --client-secret <value>
    --authorization-url <value>
    --token-url <value>
```

## Options

**`--name`** (string)

Provider name (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)
- Alias: `-n`
- Constraints: 3–50 characters; matches `^[a-zA-Z0-9_-]+$`.

**`--client-id`** (string)

OAuth2 client ID (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)

**`--client-secret`** (string)

OAuth2 client secret (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted as a secret when `--interactive` is set)

**`--authorization-url`** (string)

Authorization endpoint URL (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)

**`--token-url`** (string)

Token endpoint URL (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Create an OAuth2 provider:

```bash
grn agentbase access outbound-auth oauth2 create \
  --name my-oauth2-provider \
  --client-id abc123 \
  --client-secret s3cr3t \
  --authorization-url https://idp.example.com/oauth2/authorize \
  --token-url https://idp.example.com/oauth2/token
```

The created provider exposes a server-generated **Callback URL** (shown in the
output); register it with your IdP before requesting tokens.
