# access outbound-auth delegated create

Create a delegated API key provider.

## Description

Create a new delegated API key provider. The name must be 3-50 characters and match
the pattern `^[a-zA-Z0-9_-]+$`.

## Synopsis

```text
grn agentbase access outbound-auth delegated create
    --name <value>
```

## Options

**`--name`** (string)

Provider name (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)
- Alias: `-n`
- Constraints: 3–50 characters; matches `^[a-zA-Z0-9_-]+$`.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Create a delegated API key provider:

```bash
grn agentbase access outbound-auth delegated create --name my-delegated-provider
```
