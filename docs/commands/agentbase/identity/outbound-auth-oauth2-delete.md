# identity outbound-auth oauth2 delete

Delete an OAuth2 provider.

## Description

Delete an OAuth2 provider by name.

## Synopsis

```text
grn agentbase identity outbound-auth oauth2 delete <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Delete an OAuth2 provider:

```bash
grn agentbase identity outbound-auth oauth2 delete my-oauth2-provider
```
