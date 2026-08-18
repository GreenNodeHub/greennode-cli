# access agent-id use

Set the current agent identity.

## Description

Set the given agent identity name as the current identity in the shared `~/.greennode` profile (per-profile).

## Synopsis

```text
grn agentbase access agent-id use <name>
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Set the current agent identity:

```bash
grn agentbase access agent-id use my-agent
```
