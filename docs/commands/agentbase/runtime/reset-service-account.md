# runtime reset-service-account

Reset a runtime's IAM service account.

## Description

Rotate the runtime's IAM service account
(POST `/agent-runtimes/{id}/reset-service-account`). No body; 200 OK. The
service also exposes a deprecated PATCH variant; this POST is canonical and the
PATCH QC row maps to it.

## Synopsis

```text
grn agentbase runtime reset-service-account <id>
```

## Arguments

**`<id>`** (string) — runtime id. Required.

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase runtime reset-service-account rt-1
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`
