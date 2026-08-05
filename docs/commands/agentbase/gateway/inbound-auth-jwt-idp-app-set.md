# gateway inbound-auth jwt idp-app set

Set the inbound-auth JWT IdP app credentials.

## Description

Set the gateway's inbound-auth JWT IdP application credentials — the OAuth2
client/secret the gateway uses to talk to the IdP
(PUT `/api/v1/gateways/{name}/inbound-auth/jwt/idp-app`). 204 No Content on
success. Omit `--client-secret` to preserve the existing secret; a non-empty
value replaces it. The secret stays server-side after it is set.

## Synopsis

```text
grn agentbase gateway inbound-auth jwt idp-app set <name>
    --client-id <value>
    [--client-secret <value>]
    [--scope <value> ...]
```

## Arguments

**`<name>`** (string) — gateway name. Required.

## Options

**`--client-id`** (string) — IdP app client id (required).
**`--client-secret`** (string) — IdP app client secret (omit to preserve existing).
**`--scope`** (string, repeatable) — IdP app scope.

## Examples

```bash
grn agentbase gateway inbound-auth jwt idp-app set my-gw --client-id cid --client-secret s3cret --scope openid
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
