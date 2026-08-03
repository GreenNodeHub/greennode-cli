# gateway delete

Delete a gateway.

## Description

Delete a gateway by name. The gateway is deleted asynchronously; this command
returns as soon as the service accepts the deletion (state `WAITING_DELETING`).
Confirm with `grn agentbase gateway wait <name>` (which terminates on
`DELETED`).

## Synopsis

```text
grn agentbase gateway delete <name>
```

## Arguments

**`<name>`** (string)

Name of the gateway to delete.

- Required: Yes (exactly one positional argument)

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Delete a gateway and wait for it to finish:

```bash
grn agentbase gateway delete my-gateway
grn agentbase gateway wait my-gateway
```
