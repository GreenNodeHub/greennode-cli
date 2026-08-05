# gateway inbound-auth jwt idp-app clear

Clear the inbound-auth JWT IdP app credentials.

## Description

Clear the gateway's inbound-auth JWT IdP application credentials
(DELETE `/api/v1/gateways/{name}/inbound-auth/jwt/idp-app`). 204 on success.

## Synopsis

```text
grn agentbase gateway inbound-auth jwt idp-app clear <name>
```

## Arguments

**`<name>`** (string) — gateway name. Required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase gateway inbound-auth jwt idp-app clear my-gw
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
