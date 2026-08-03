# cr repository image list

List images.

## Description

List images in the user's repository with paging. Table columns: Name,
Artifacts, Pulls, Updated. The paging envelope is `{data, page, pageSize,
totalItem, totalPage}`; the table footer prints `Page <n> of <total> (<total
items>)`.

Use `-o id` to print just image names, or `-o json` for the full envelope.

## Synopsis

```text
grn agentbase cr repository image list
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

Page size.

- Required: No
- Default: `10`

**`--name`** (string)

Filter by image name (case-insensitive substring).

- Required: No

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase cr repository image list
```

Filter by name and request a larger page:

```bash
grn agentbase cr repository image list --name myapp --size 50
```
