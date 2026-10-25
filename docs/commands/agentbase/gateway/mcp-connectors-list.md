# gateway mcp-connectors list

List MCP connectors (targets) across all gateways.

## Description

List every MCP connector (target) attached across your gateways, flattened
into one list (GET `/api/v1/mcp-connectors`). Each item carries a ready-to-use
`connectUrl`, the derived template type (or `CUSTOM`), the upstream
outbound-auth descriptor, and parent-gateway context including inbound-auth
mode. Filter with `--gateway-name` and `--state`.

This is the flat "connector catalog" view: use it to find a `connectUrl` to
hand to any MCP client without drilling into one gateway.

## Synopsis

```text
grn agentbase gateway mcp-connectors list
    [--gateway-name <value>]
    [--state <value>]
    [--page <value>]
    [--size <value>]
```

## Options

**`--gateway-name`** (string)

Filter by parent gateway name.

- Required: No
- Default: (empty)

**`--state`** (string)

Filter by parent gateway state (e.g. `ACTIVE`).

- Required: No
- Default: (empty)

**`--page`** (int)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (int)

Page size (max 100).

- Required: No
- Default: `50`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase gateway mcp-connectors list
grn agentbase gateway mcp-connectors list --gateway-name my-gw --state ACTIVE
grn agentbase gateway mcp-connectors list -o json --query 'items[0].connectUrl'
```
