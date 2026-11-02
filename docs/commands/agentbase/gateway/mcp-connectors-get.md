# gateway mcp-connectors get

Get one MCP connector by gateway + target name.

## Description

Get a single MCP connector (one target on one caller-owned gateway) by
composite key (GET `/api/v1/mcp-connectors/{gatewayName}/{targetName}`).
Ownership is caller-scoped, so another user's gateway looks like a 404.

The response shows the connector's `connectUrl`, template type (or `CUSTOM`),
outbound-auth descriptor, and parent gateway context including inbound-auth
mode.

## Synopsis

```text
grn agentbase gateway mcp-connectors get <gateway> <target>
```

## Arguments

**`<gateway>`** (string) — parent gateway name. Required.

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
grn agentbase gateway mcp-connectors get my-gw github
```
