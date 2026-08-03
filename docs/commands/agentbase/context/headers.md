# context headers

Show platform request headers reference.

## Description

Display the standard `X-GreenNode-AgentBase-*` HTTP request headers used by the platform.

## Synopsis

```text
grn agentbase context headers
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output` — Output format: "table", "json", or "id" (default `table`)
- `-i, --interactive` — Prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Show the platform request headers reference:

```bash
grn agentbase context headers
```
