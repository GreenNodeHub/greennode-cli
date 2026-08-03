# runtime trace get

Fetch a trace by id.

## Description

Fetch a single trace by id (GET `/agent-runtimes:get-trace?traceId=<value>`).
The endpoint is a Google-AIP custom verb that forwards arbitrary query params to
the tracing backend and returns that backend's raw JSON, printed verbatim. Use
`--param key=value` (repeatable) to pass params through.

## Synopsis

```text
grn agentbase runtime trace get <trace-id>
    [--param <key=value> ...]
```

## Arguments

**`<trace-id>`** (string) — trace id (the required `traceId` query param).

## Options

**`--param`** (string, repeatable) — passthrough query param `key=value`.

## Examples

```bash
grn agentbase runtime trace get abc-123 --param service=runtime
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
