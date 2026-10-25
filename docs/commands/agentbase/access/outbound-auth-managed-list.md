# outbound-auth managed list

List platform-managed OAuth2 providers.

## Description

Retrieve a paginated list of platform-managed OAuth2 providers (GET
`/api/v1/outbound-auth/managed-oauth2-providers`). Managed providers keep
their credentials server-side (never exposed), so the CLI surface is
read-only. Each provider has a fixed `callbackUrl` the delegated browser flow
returns to.

## Synopsis

```text
grn agentbase access outbound-auth managed list
    [--page <value>]
    [--size <value>]
```

## Options

**`--page`** (int)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (int)

Page size.

- Required: No
- Default: `20`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase access outbound-auth managed list
grn agentbase access outbound-auth managed list --size 50
```
