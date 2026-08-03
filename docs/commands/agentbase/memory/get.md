# memory get

Show a memory.

## Description

Show details of a single memory by ID — ID, name, description, event TTL, state, created time, and updated time.

## Synopsis

```text
grn agentbase memory get <id>
```

Positional argument:

- `<id>` — the memory ID. Exactly one argument is required.

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Show a memory:

```bash
grn agentbase memory get mem-123
```
