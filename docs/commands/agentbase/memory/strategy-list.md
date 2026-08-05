# memory strategy list

List a memory's long-term-memory strategies.

## Description

List the long-term-memory strategies configured on a memory
(GET `/memories/{id}/long-term-memory-strategies`). Each row shows the
strategy id, name, type, namespace template, status, and timestamps.

## Synopsis

```text
grn agentbase memory strategy list <id>
```

## Arguments

**`<id>`** (string) — memory id. Required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase memory strategy list mem-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
