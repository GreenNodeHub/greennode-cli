# identity outbound-auth static delete

Delete a static API key provider.

## Description

Delete a static API key provider by name.

## Synopsis

```text
grn agentbase identity outbound-auth static delete <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Delete a static API key provider:

```bash
grn agentbase identity outbound-auth static delete my-apikey-provider
```
