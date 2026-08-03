# policy group policy list

List policies within a group.

## Description

List the policies (rules) inside a policy group (max 10 per group). Supports
pagination and a case-insensitive name-substring filter.

The query is sent as snake_case `?page=&page_size=&name=` (so `--size` maps to
`page_size`), but the response field is camelCase `pageSize`.

## Synopsis

```text
grn agentbase policy group policy list <group-id>
    [--page <value>]
    [--size <value>]
    [--name <value>]
```

Positional arguments:

- `<group-id>` — the policy group id (exactly 1 required).

## Options

**`--page`** (integer)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (integer)

Page size (1-100).

- Required: No
- Default: `10`
- Constraints: 1-100
- Sent as the `page_size` query parameter.

**`--name`** (string)

Filter by name (case-insensitive substring).

- Required: No

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

List policies in a group:

```bash
grn agentbase policy group policy list pg-abc123
```

Filter by name and request a larger page:

```bash
grn agentbase policy group policy list pg-abc123 --name allow --size 50
```

Emit IDs only (scripting):

```bash
grn agentbase policy group policy list pg-abc123 -o id
```
