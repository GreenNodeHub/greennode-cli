# cr repository artifact list

List artifacts within an image.

## Description

List artifacts (digests) within an image, with paging. `--image-name` selects
the image; `--name` filters by digest or tag (case-insensitive substring).
Table columns: Digest, Type, Size, Tags, Pushed.

The paging envelope is `{data, page, pageSize, totalItem, totalPage}`. Use
`-o id` to print just digests, or `-o json` for the full envelope.

## Synopsis

```text
grn agentbase cr repository artifact list
    --image-name <value>
    [--page <value>]
    [--size <value>]
    [--name <value>]
```

## Options

**`--image-name`** (string)

Image name (required).

- Required: Yes

**`--page`** (integer)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (integer)

Page size.

- Required: No
- Default: `10`

**`--name`** (string)

Filter by digest/tag (case-insensitive substring).

- Required: No

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase cr repository artifact list --image-name my-agent
```
