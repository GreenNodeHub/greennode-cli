# memory delete

Delete a memory.

## Description

Delete a memory by ID. Deletion is a soft delete: the memory transitions `ACTIVE → DELETED` rather than being permanently removed.

## Synopsis

```text
grn agentbase memory delete <id>
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

Delete a memory:

```bash
grn agentbase memory delete mem-123
```
