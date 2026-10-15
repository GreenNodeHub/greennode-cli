# tools code-interpreter list

List code interpreters.

## Description

List the platform-managed code-execution sandboxes
(GET `/v1/code-interpreters`), 1-based paginated.

## Synopsis

```text
grn agentbase tools code-interpreter list
    [--page <value>]
    [--size <value>]
```

## Options

**`--page`** (integer) — page number (1-based, validated client-side).
Default `1`.

**`--size`** (integer) — page size (must be ≥ 1, validated client-side).
Default `10`.

## Output

- `table` — ID, Name, Description, Status, Created + a paging footer
- `json` — the raw `{listData, page, pageSize, totalItem, totalPage}` envelope
- `id` — the first code interpreter's id

## Examples

```bash
grn agentbase tools code-interpreter list
grn agentbase tools code-interpreter list --page 2 --size 20
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
