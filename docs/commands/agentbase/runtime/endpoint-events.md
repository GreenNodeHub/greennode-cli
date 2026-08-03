# runtime endpoint events

List an endpoint's kubernetes events.

## Description

List kubernetes events for an endpoint
(GET `/agent-runtimes/{id}/endpoints/{endpointId}/events`). Each row carries a
message and lastTimestamp.

## Synopsis

```text
grn agentbase runtime endpoint events <id> <endpoint-id>
```

## Arguments

**`<id>`** / **`<endpoint-id>`** — runtime / endpoint ids. Both required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase runtime endpoint events rt-1 ep-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
