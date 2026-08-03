# gateway service-account repair

Repair a gateway's IAM service account.

## Description

Trigger an IAM service-account repair for a gateway
(POST `/api/v1/gateways/{name}/service-account/repair`). Use when
`iam.lastAuthFailureAt` is set — the gateway could not exchange for a token and
needs its service account re-issued. No body; returns the refreshed gateway.

## Synopsis

```text
grn agentbase gateway service-account repair <name>
```

## Arguments

**`<name>`** (string) — gateway name. Required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase gateway service-account repair my-gw
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
