# runtime trace tag-values

List distinct values for a trace tag.

## Description

List distinct values for a trace tag
(GET `/agent-runtimes:trace-search-tag-values?tagKey=<value>`). Forwards
arbitrary extra query params to the tracing backend; the backend's raw JSON is
printed verbatim. Use `--param key=value` (repeatable) to pass params through.

## Synopsis

```text
grn agentbase runtime trace tag-values --tag-key <value>
    [--param <key=value> ...]
```

## Options

**`--tag-key`** (string) — tag key (required).
**`--param`** (string, repeatable) — passthrough query param `key=value`.

## Examples

```bash
grn agentbase runtime trace tag-values --tag-key env
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
