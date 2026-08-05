# catalog openclaw delete

Delete an OpenClaw workspace.

## Description

Delete an OpenClaw workspace by id (DELETE `/v1/openclaws/{id}`). The service
returns 200 with no body; the command prints the deleted id.

## Synopsis

```text
grn agentbase catalog openclaw delete <id>
```

## Arguments

**`<id>`** (string)

Id of the OpenClaw workspace to delete.

- Required: Yes (exactly one positional argument)

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase catalog openclaw delete oc-123
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
