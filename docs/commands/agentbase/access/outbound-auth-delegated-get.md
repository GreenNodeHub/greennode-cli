# access outbound-auth delegated get

Get a delegated API key provider.

## Description

Retrieve a delegated API key provider by name.

## Synopsis

```text
grn agentbase access outbound-auth delegated get <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Get a delegated API key provider:

```bash
grn agentbase access outbound-auth delegated get my-delegated-provider
```

View the full record as JSON:

```bash
grn agentbase access outbound-auth delegated get my-delegated-provider -o json
```
