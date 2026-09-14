# access agent-id get

Get an agent identity by name.

## Description

Retrieve a specific agent identity by its name.

## Synopsis

```text
grn agentbase access agent-id get <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Get an identity:

```bash
grn agentbase access agent-id get my-agent
```

View the full record as JSON:

```bash
grn agentbase access agent-id get my-agent -o json
```
