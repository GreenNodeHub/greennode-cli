# runtime trace search

Search traces.

## Description

Search traces (GET `/agent-runtimes:search-traces`), forwarding arbitrary
query params to the tracing backend. The backend's raw JSON is printed verbatim.
Use `--param key=value` (repeatable) to pass params through.

## Synopsis

```text
grn agentbase runtime trace search
    [--param <key=value> ...]
```

## Arguments

None.

## Options

**`--param`** (string, repeatable) — passthrough query param `key=value`.

## Examples

```bash
grn agentbase runtime trace search --param service=runtime --param limit=20
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
