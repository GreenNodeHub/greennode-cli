# memory session list

List an actor's sessions on a memory.

## Description

List the sessions recorded for an actor on a memory
(GET `/memories/{id}/actors/{actorId}/sessions`), 1-based paginated.

## Synopsis

```text
grn agentbase memory session list <id> <actor-id>
    [--page <value>]
    [--size <value>]
```

## Arguments

**`<id>`** (string) — memory id. Required.
**`<actor-id>`** (string) — actor id. Required.

## Options

**`--page`** (integer) — page number (1-based). Default `1`.
**`--size`** (integer) — page size. Default `10`.

## Examples

```bash
grn agentbase memory session list mem-1 act-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
