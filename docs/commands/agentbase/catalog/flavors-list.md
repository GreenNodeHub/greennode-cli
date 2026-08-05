# catalog flavors list

List runtime placement flavors.

## Description

List runtime placement flavors served by the runtime service catalog
(GET `/v1/flavors`). These are the flavors selectable when creating an agent
runtime. Filters are optional; omit them to list every flavor.

## Synopsis

```text
grn agentbase catalog flavors list
    [--resource-type <value>]
```

## Options

**`--resource-type`** (string)

Filter by supported resource type.

- Required: No

## Examples

List every flavor:

```bash
grn agentbase catalog flavors list
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
