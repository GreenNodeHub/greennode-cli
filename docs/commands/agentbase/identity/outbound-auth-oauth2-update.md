# identity outbound-auth oauth2 update

Update an OAuth2 provider.

## Description

Update an existing OAuth2 provider. All of client-id, client-secret,
authorization-url, and token-url are required.

## Synopsis

```text
grn agentbase identity outbound-auth oauth2 update <name>
    --client-id <value>
    --client-secret <value>
    --authorization-url <value>
    --token-url <value>
```

## Options

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

Rotate all four endpoints/credentials on an existing provider:

```bash
grn agentbase identity outbound-auth oauth2 update my-oauth2-provider \
  --client-id abc123 \
  --client-secret new-s3cr3t \
  --authorization-url https://idp.example.com/oauth2/authorize \
  --token-url https://idp.example.com/oauth2/token
```
