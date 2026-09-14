# access outbound-auth oauth2 list

List OAuth2 providers.

## Description

Retrieve a paginated list of OAuth2 providers.

## Synopsis

```text
grn agentbase access outbound-auth oauth2 list
    [--page <value>]
    [--size <value>]
```

## Options

**`--page`** (integer)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (integer)

Page size.

- Required: No
- Default: `20`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

List OAuth2 providers:

```bash
grn agentbase access outbound-auth oauth2 list
```

List the second page with 50 per page:

```bash
grn agentbase access outbound-auth oauth2 list --page 2 --size 50
```
