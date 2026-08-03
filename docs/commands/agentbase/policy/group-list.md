# policy group list

List policy groups.

## Description

List the policy groups owned by the authenticated user (max 20). Supports
pagination and a case-insensitive name-substring filter.

The query is sent as snake_case `?page=&page_size=&name=` (so `--size` maps to
`page_size`), but the response field is camelCase `pageSize`.

## Synopsis

```text
grn agentbase policy group list
    [--page <value>]
    [--size <value>]
    [--name <value>]
```

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

List the first page of groups:

```bash
grn agentbase policy group list
```

Filter by name and request a larger page:

```bash
grn agentbase policy group list --name prod --size 50
```

Emit IDs only (scripting):

```bash
grn agentbase policy group list -o id
```
