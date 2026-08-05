# runtime list

List agent runtimes.

## Description

List agent runtimes. The table output shows the runtime's `ID`, `Name`, `State`,
and `Created` columns. Use [get](get.md) for the full spec of a single runtime.

## Synopsis

```text
grn agentbase runtime list
    [--page <value>]
    [--size <value>]
```

## Options

**`--page`** (int)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (int)

Page size.

- Required: No
- Default: `10`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

List the first page of runtimes:

```bash
grn agentbase runtime list
```

List a larger page:

```bash
grn agentbase runtime list --size 50
```

Emit the raw JSON response (full paging metadata):

```bash
grn agentbase runtime list -o json
```
