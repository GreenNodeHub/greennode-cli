# identity workload delete

Delete an agent identity.

## Description

Delete an agent identity by name. The identity will be soft-deleted.

## Synopsis

```text
grn agentbase identity workload delete <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Delete an identity:

```bash
grn agentbase identity workload delete my-agent
```
