# identity workload list

List agent identities.

## Description

Retrieve a paginated list of all agent identities owned by the authenticated user.

## Synopsis

```text
grn agentbase identity workload list
    [--page <value>]
    [--size <value>]
```

## Options

**`--page`** (integer)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (integer)

Page size.

- Required: No
- Default: `20`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

List identities (default page/size):

```bash
grn agentbase identity workload list
```

List the second page with 50 per page:

```bash
grn agentbase identity workload list --page 2 --size 50
```

Emit IDs only (scripting):

```bash
grn agentbase identity workload list -o id
```
