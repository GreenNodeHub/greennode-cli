# context current

Show the active environment and resolved endpoints.

## Description

Display the currently active environment (resolved from the profile's `iam_env`) and all agentbase API base URLs.

The environment selects the agentbase API endpoints AND the IAM v2 token URL; it is stored as `iam_env` in the shared `~/.greennode` profile (default `prod`), the same selector vks/vserver use. Switch it with `grn configure set iam_env <dev|prod>` (machine) or `grn login --iam-env <env>` (user).

When `--endpoint-url` is set, the six service endpoints (Identity/Runtime/Memory/Gateway/Policy/CR) reflect the override — host swapped, each service's path kept. The `OAuth2 Token` URL is shown unchanged: it is resolved from `iam_env` by the auth provider, not overridden by `--endpoint-url`.

## Synopsis

```text
grn agentbase context current
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output` — Output format: "table", "json", or "id" (default `table`)
- `-i, --interactive` — Prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Show the active environment and resolved endpoints:

```bash
grn agentbase context current
```
