# gateway list

List gateways.

## Description

List gateways in the active profile/environment, paginated. The table view shows
Name, Mode, State, Flavor, Replicas, Endpoint, and Created per gateway, and a
page summary on stderr.

## Synopsis

```text
grn agentbase gateway list
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
- Default: `50`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

List the first page of gateways:

```bash
grn agentbase gateway list
```

List a specific page at a smaller page size, as JSON:

```bash
grn agentbase gateway list --page 2 --size 20 -o json
```
