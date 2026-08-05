# memory record list

List a memory's long-term records.

## Description

List the long-term memory records under a namespace
(GET `/memories/{id}/memory-records` with `?namespace=&limit=`). The
record schema uses snake_case json keys (`created_at`, `updated_at`).

## Synopsis

```text
grn agentbase memory record list <id> --namespace <value> [--limit <value>]
```

## Arguments

**`<id>`** (string) — memory id. Required.

## Options

**`--namespace`** (string) — resolved namespace (required).
**`--limit`** (integer) — max results. Default `100`.

## Examples

```bash
grn agentbase memory record list mem-1 --namespace /strategies/SEMANTIC/actors/act-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
