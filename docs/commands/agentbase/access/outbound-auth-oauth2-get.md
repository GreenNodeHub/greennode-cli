# access outbound-auth oauth2 get

Get an OAuth2 provider.

## Description

Retrieve an OAuth2 provider by name.

## Synopsis

```text
grn agentbase access outbound-auth oauth2 get <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Get an OAuth2 provider:

```bash
grn agentbase access outbound-auth oauth2 get my-oauth2-provider
```

View the full record as JSON:

```bash
grn agentbase access outbound-auth oauth2 get my-oauth2-provider -o json
```
