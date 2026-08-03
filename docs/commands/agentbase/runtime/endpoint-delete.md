# runtime endpoint delete

Delete a runtime endpoint.

## Description

Delete an endpoint (DELETE `/agent-runtimes/{id}/endpoints/{endpointId}`).
Returns 200 with the deleted endpoint, or an empty body; the command falls back
to printing the deleted id when the body is empty.

## Synopsis

```text
grn agentbase runtime endpoint delete <id> <endpoint-id>
```

## Arguments

**`<id>`** (string) — runtime id. Required.
**`<endpoint-id>`** (string) — endpoint id. Required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase runtime endpoint delete rt-1 ep-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
