# outbound-auth managed get

Get a platform-managed OAuth2 provider.

## Description

Retrieve a platform-managed OAuth2 provider summary by name (GET
`/api/v1/outbound-auth/managed-oauth2-providers/{name}`). Shows the
provider's id, name, status, and fixed `callbackUrl`; credentials are never
exposed.

## Synopsis

```text
grn agentbase access outbound-auth managed get <name>
```

## Arguments

**`<name>`** (string) — managed provider name. Required.

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase access outbound-auth managed get google
```
