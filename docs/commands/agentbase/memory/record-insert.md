# memory record insert

Insert memory records directly.

## Description

Insert one or more long-term memory records directly, bypassing generation
(POST `/memories/{id}/memory-records:insert-directly` with
`?namespace=<value>` and a body of `{memoryRecords: [...]}`). `--record` is
repeatable; at least one is required. 200 OK on success.

## Synopsis

```text
grn agentbase memory record insert <id> --namespace <value> --record <value> [--record <value> ...]
```

## Arguments

**`<id>`** (string) — memory id. Required.

## Options

**`--namespace`** (string) — resolved namespace (required).
**`--record`** (string, repeatable) — record text (at least one required).

## Examples

```bash
grn agentbase memory record insert mem-1 --namespace /strategies/SEMANTIC/actors/act-1 --record "prefers dark mode"
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
