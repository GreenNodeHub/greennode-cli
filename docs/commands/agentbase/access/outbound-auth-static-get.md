# access outbound-auth static get

Get a static API key provider.

## Description

Retrieve a static API key provider by name.

## Synopsis

```text
grn agentbase access outbound-auth static get <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Get a static API key provider:

```bash
grn agentbase access outbound-auth static get my-apikey-provider
```

View the full record as JSON:

```bash
grn agentbase access outbound-auth static get my-apikey-provider -o json
```
