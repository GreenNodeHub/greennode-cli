# memory record delete

Delete a memory record.

## Description

Delete a single long-term memory record
(DELETE `/memories/{id}/memory-records/{memoryRecordId}`). 200 OK; prints the
deleted record id.

## Synopsis

```text
grn agentbase memory record delete <id> <record-id>
```

## Arguments

**`<id>`** (string) — memory id. Required.
**`<record-id>`** (string) — memory record id. Required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase memory record delete mem-1 rec-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
