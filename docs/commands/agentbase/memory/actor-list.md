# memory actor list

List a memory's actors.

## Description

List the actors recorded against a memory (GET `/memories/{id}/actors`),
1-based paginated. Each row shows the memory id, actor id, and actor status.

## Synopsis

```text
grn agentbase memory actor list <id>
    [--page <value>]
    [--size <value>]
```

## Arguments

**`<id>`** (string)

Memory id. Required.

## Options

**`--page`** (integer) — page number (1-based). Default `1`.
**`--size`** (integer) — page size. Default `10`.

## Examples

```bash
grn agentbase memory actor list mem-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
