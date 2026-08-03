# runtime endpoint stop

Stop a runtime endpoint.

## Description

Stop an endpoint (POST `/agent-runtimes/{id}/endpoints/{endpointId}/stop`).
No body; 200 OK.

## Synopsis

```text
grn agentbase runtime endpoint stop <id> <endpoint-id>
```

## Arguments

**`<id>`** / **`<endpoint-id>`** — runtime / endpoint ids. Both required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase runtime endpoint stop rt-1 ep-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
