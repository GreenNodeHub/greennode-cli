# gateway flavors list

List gateway placement flavors.

## Description

List gateway placement flavors (GET `/api/v1/flavors`). These are the flavors
selectable as a gateway's `flavorId` — distinct from the runtime compute-flavor
catalog. Filters are optional.

## Synopsis

```text
grn agentbase gateway flavors list
    [--resource-type <value>]
    [--network-mode <value>]
    [--zone-id <value>]
```

## Options

**`--resource-type`** (string) — filter by resource type.
**`--network-mode`** (string) — filter by network mode (PUBLIC|PRIVATE).
**`--zone-id`** (string) — filter by zone id.

## Examples

```bash
grn agentbase gateway flavors list --network-mode PRIVATE -o json
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
