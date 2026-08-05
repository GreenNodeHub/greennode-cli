# catalog openclaw stop

Stop an OpenClaw workspace.

## Description

Stop an OpenClaw workspace by id (POST `/v1/openclaws/{id}/stop`). No body;
200 OK on success.

## Synopsis

```text
grn agentbase catalog openclaw stop <id>
```

## Arguments

**`<id>`** (string)

Id of the OpenClaw workspace to stop.

- Required: Yes (exactly one positional argument)

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase catalog openclaw stop oc-123
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
