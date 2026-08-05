# memory list

List memories.

## Description

List memories, paginated. In table format each row shows ID, name, state, event TTL, and created time; the current page, total pages, and total item count are printed on stderr. With no results, prints "No memories found." on stderr.

## Synopsis

```text
grn agentbase memory list
    [--page <number>]
    [--size <number>]
```

## Options

**`--page`** (integer)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (integer)

Page size.

- Required: No
- Default: `10`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

List the first page of memories:

```bash
grn agentbase memory list
```

List a larger page:

```bash
grn agentbase memory list --page 2 --size 50
```

Emit IDs only (scripting):

```bash
grn agentbase memory list -o id
```
