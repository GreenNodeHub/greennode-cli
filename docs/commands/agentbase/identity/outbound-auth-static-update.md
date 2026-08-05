# identity outbound-auth static update

Update a static API key provider.

## Description

Update the API key value of an existing static API key provider.

## Synopsis

```text
grn agentbase identity outbound-auth static update <name>
    --apikey <value>
```

## Options

**`--apikey`** (string)

New API key value (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted as a secret when `--interactive` is set)

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Rotate the API key on an existing provider:

```bash
grn agentbase identity outbound-auth static update my-apikey-provider \
  --apikey sk-newvalue
```
