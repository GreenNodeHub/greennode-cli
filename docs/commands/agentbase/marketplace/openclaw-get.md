# marketplace openclaw get

Show an OpenClaw workspace.

## Description

Fetch and display a single OpenClaw workspace by id (GET `/v1/openclaws/{id}`),
rendered as a detail table (or JSON/id per the global output format).

## Synopsis

```text
grn agentbase marketplace openclaw get <id>
```

## Arguments

**`<id>`** (string)

Id of the OpenClaw workspace to show.

- Required: Yes (exactly one positional argument)

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase marketplace openclaw get oc-123 -o json
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
