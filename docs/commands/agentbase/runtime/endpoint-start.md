# runtime endpoint start

Start a runtime endpoint.

## Description

Start an endpoint (POST `/agent-runtimes/{id}/endpoints/{endpointId}/start`).
No body; 200 OK.

## Synopsis

```text
grn agentbase runtime endpoint start <id> <endpoint-id>
```

## Arguments

**`<id>`** / **`<endpoint-id>`** — runtime / endpoint ids. Both required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase runtime endpoint start rt-1 ep-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
