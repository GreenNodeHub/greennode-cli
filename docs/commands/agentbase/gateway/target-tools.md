# gateway target tools

Show a target's last-synced tools/list.

## Description

Get a target's last-synced tools/list (GET
`/api/v1/gateways/{name}/targets/{target}/tools`). This is the persisted
snapshot; refresh it with [target sync](target-sync.md).

## Synopsis

```text
grn agentbase gateway target tools <gateway> <target>
```

## Arguments

**`<gateway>`** (string) — gateway name. Required.

**`<target>`** (string) — target name on that gateway. Required.

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase gateway target tools my-gw github
```
