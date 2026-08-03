# policy group get

Show a policy group.

## Description

Fetch a single policy group by id and print its detail (id, name, description,
created/updated timestamps).

## Synopsis

```text
grn agentbase policy group get <group-id>
```

Positional arguments:

- `<group-id>` — the policy group id (exactly 1 required).

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Show a group:

```bash
grn agentbase policy group get pg-abc123
```
