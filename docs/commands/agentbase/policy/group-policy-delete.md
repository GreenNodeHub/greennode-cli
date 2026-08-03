# policy group policy delete

Delete a policy.

## Description

Delete a single policy by id within a group.

## Synopsis

```text
grn agentbase policy group policy delete <group-id> <policy-id>
```

Positional arguments:

- `<group-id>` — the policy group id (exactly 1 required).
- `<policy-id>` — the policy id (exactly 1 required).

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Delete a policy:

```bash
grn agentbase policy group policy delete pg-abc123 pol-def456
```
