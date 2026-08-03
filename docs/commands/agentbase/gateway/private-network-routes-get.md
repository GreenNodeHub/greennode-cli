# gateway private-network routes get

Show a PRIVATE-mode gateway's private-network routes.

## Description

Show the CIDR routes programmed on a PRIVATE-mode gateway's worker nodes
(GET `/api/v1/gateways/{name}/private-network/routes`). A PUBLIC-mode gateway
404s with `private_network_not_applicable`, which is surfaced as-is.

## Synopsis

```text
grn agentbase gateway private-network routes get <name>
```

## Arguments

**`<name>`** (string) — gateway name. Required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase gateway private-network routes get my-gw
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
