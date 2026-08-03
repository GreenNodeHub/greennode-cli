# policy group delete

Delete a policy group (cascades to its policies).

## Description

Delete a policy group by id. Deletion cascades to the policies contained in the
group.

## Synopsis

```text
grn agentbase policy group delete <group-id>
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

Delete a group:

```bash
grn agentbase policy group delete pg-abc123
```
